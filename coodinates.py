import os
import subprocess
from datetime import datetime
from geopy.geocoders import Nominatim
from geopy.exc import GeocoderTimedOut, GeocoderServiceError
import pymongo
import json
import time
import logging

# Set up logging
logging.basicConfig(
    filename='station_coordinates_update.log',
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)

def create_backup():
    backup_dir = "database_backups"
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    backup_name = f"train_database_backup_{timestamp}"
    
    try:
        # Create backup directory
        os.makedirs(backup_dir, exist_ok=True)
        backup_path = os.path.join(backup_dir, backup_name)
        
        # Create compressed backup
        archive_path = f"{backup_path}.gz"
        subprocess.run([
            "mongodump",
            "--db=train_database",
            f"--archive={archive_path}",
            "--gzip"
        ], check=True)
        
        logging.info(f"Database backup created successfully at: {archive_path}")
        return True
    except Exception as e:
        logging.error(f"Backup failed: {str(e)}")
        return False

def get_station_coordinates(station_code, station_name, retry_count=3):
    geolocator = Nominatim(user_agent="indian_railways_app")
    
    # List of search patterns to try
    search_patterns = [
        f"{station_name} Railway Station, India",
        f"{station_code} Railway Station, India",
        f"{station_name} Junction, India",
        f"{station_name}, India"
    ]
    
    for pattern in search_patterns:
        for attempt in range(retry_count):
            try:
                location = geolocator.geocode(pattern, timeout=10)
                if location:
                    return {
                        "lat": location.latitude,
                        "lon": location.longitude,
                        "source": pattern
                    }
            except (GeocoderTimedOut, GeocoderServiceError):
                if attempt < retry_count - 1:
                    time.sleep(2)  # Wait before retry
                continue
            except Exception as e:
                logging.error(f"Error geocoding {station_code} ({pattern}): {str(e)}")
                break
            
            time.sleep(1)  # Rate limiting between patterns
    
    return None

def update_station_coordinates():
    try:
        # Take backup first
        if not create_backup():
            logging.error("Backup failed, aborting coordinate update")
            return False
        
        # Connect to MongoDB
        client = pymongo.MongoClient("mongodb+srv://arshadnoor585:Dap4k3TMe8XKRzwi@traindata.heu7f.mongodb.net/?retryWrites=true&w=majority&appName=traindata")
        db = client["train_database"]
        
        # Get all unique stations
        stations = set()
        station_names = {}  # Map codes to names
        
        for train in db.trains.find({}, {"schedule_table.station": 1}):
            for stop in train.get("schedule_table", []):
                station_full = stop.get("station", "")
                parts = station_full.split(" - ")
                if len(parts) >= 2:
                    code = parts[0].strip()
                    name = parts[1].strip()
                    stations.add(code)
                    station_names[code] = name
                elif station_full:
                    stations.add(station_full)
                    station_names[station_full] = station_full

        # Create or update coordinates collection
        coords_collection = db.station_coordinates
        
        # Create index on station_code if it doesn't exist
        coords_collection.create_index("station_code", unique=True)
        
        # Process each station
        total_stations = len(stations)
        processed = 0
        success = 0
        failed = 0
        
        logging.info(f"Starting coordinate update for {total_stations} stations")
        
        for station_code in stations:
            processed += 1
            station_name = station_names.get(station_code, station_code)
            
            # Check if coordinates already exist
            existing = coords_collection.find_one({"station_code": station_code})
            if existing and existing.get("coordinates"):
                success += 1
                continue
                
            coords = get_station_coordinates(station_code, station_name)
            
            if coords:
                try:
                    coords_collection.update_one(
                        {"station_code": station_code},
                        {
                            "$set": {
                                "station_name": station_name,
                                "coordinates": coords,
                                "updated_at": datetime.now()
                            }
                        },
                        upsert=True
                    )
                    success += 1
                    logging.info(f"Updated coordinates for {station_code} ({processed}/{total_stations})")
                except Exception as e:
                    failed += 1
                    logging.error(f"Failed to update DB for {station_code}: {str(e)}")
            else:
                failed += 1
                logging.warning(f"Could not find coordinates for {station_code}")
            
            # Progress update every 10 stations
            if processed % 10 == 0:
                logging.info(f"Progress: {processed}/{total_stations} (Success: {success}, Failed: {failed})")
            
            time.sleep(1)  # Rate limiting
        
        # Save summary to file
        summary = {
            "total_stations": total_stations,
            "successful_updates": success,
            "failed_updates": failed,
            "timestamp": datetime.now().isoformat()
        }
        
        with open("coordinate_update_summary.json", "w") as f:
            json.dump(summary, f, indent=2)
        
        logging.info("Coordinate update completed")
        logging.info(f"Summary: Total={total_stations}, Success={success}, Failed={failed}")
        
        return True
        
    except Exception as e:
        logging.error(f"Coordinate update failed: {str(e)}")
        return False

if __name__ == "__main__":
    logging.info("Starting station coordinates update process")
    if update_station_coordinates():
        logging.info("Process completed successfully")
    else:
        logging.error("Process failed")