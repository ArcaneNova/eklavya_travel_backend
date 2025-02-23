package models

type Village struct {
    Title           string          `json:"title"`
    Address         string          `json:"address"`
    State           string          `json:"state"`
    District        string          `json:"district"`
    Subdistrict     string          `json:"subdistrict"`
    Village         string          `json:"village"`
    Latitude        float64         `json:"latitude"`
    Longitude       float64         `json:"longitude"`
    CollegesNear    []CollegeNear   `json:"colleges_near"`
    SchoolsNear     []SchoolNear    `json:"schools_near"`
    NationalHighways []Highway       `json:"national_highways"`
    Rivers          []River         `json:"rivers"`
    MainVillage     string          `json:"main_village"`
}

type CollegeNear struct {
    Name    string `json:"name"`
    Address string `json:"address"`
}

type SchoolNear struct {
    Name    string `json:"name"`
    Address string `json:"address"`
}

type Highway struct {
    Highway string `json:"highway"`
}

type NearbyFacility struct {
    Title     string  `json:"title"`
    Address   string  `json:"address"`
    Distance  float64 `json:"distance"`
    Latitude  float64 `json:"latitude"`
    Longitude float64 `json:"longitude"`
}

type VillageDetails struct {
    BasicInfo        Village         `json:"basic_info"`
    NearbyFacilities struct {
        ATMs           []NearbyFacility `json:"atms"`
        BusStops       []NearbyFacility `json:"bus_stops"`
        Cinemas        []NearbyFacility `json:"cinemas"`
        Colleges       []NearbyFacility `json:"colleges"`
        Electronics    []NearbyFacility `json:"electronics"`
        Governments    []NearbyFacility `json:"governments"`
        Hospitals      []NearbyFacility `json:"hospitals"`
        Hotels         []NearbyFacility `json:"hotels"`
        Mosques        []NearbyFacility `json:"mosques"`
        Parks          []NearbyFacility `json:"parks"`
        PetrolPumps    []NearbyFacility `json:"petrol_pumps"`
        PoliceStations []NearbyFacility `json:"police_stations"`
        Restaurants    []NearbyFacility `json:"restaurants"`
        Schools        []NearbyFacility `json:"schools"`
        Supermarkets   []NearbyFacility `json:"supermarkets"`
        Temples        []NearbyFacility `json:"temples"`
    } `json:"nearby_facilities"`
    CensusData       *CensusData     `json:"census_data,omitempty"`
}