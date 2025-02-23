package models

type Mandal struct {
    ID             string   `json:"id" bson:"_id"`
    Name           string   `json:"name" bson:"name"`
    Latitude       float64  `json:"latitude" bson:"latitude"`
    Longitude      float64  `json:"longitude" bson:"longitude"`
    Language       string   `json:"language" bson:"language"`
    Elevation      float64  `json:"elevation" bson:"elevation"`
    TelephoneCode string   `json:"telephone_code" bson:"telephone_code"`
    VehicleReg    string   `json:"vehicle_registration" bson:"vehicle_registration"`
    RTOOffice     string   `json:"rto_office" bson:"rto_office"`
    
    Assembly struct {
        Constituency string `json:"constituency" bson:"constituency"`
        MLA         string `json:"mla" bson:"mla"`
        Party       string `json:"party" bson:"party"`
        Term        string `json:"term" bson:"term"`
    } `json:"assembly" bson:"assembly"`
    
    LokSabha struct {
        Constituency string `json:"constituency" bson:"constituency"`
        MP          string `json:"mp" bson:"mp"`
        Party       string `json:"party" bson:"party"`
        Term        string `json:"term" bson:"term"`
    } `json:"lok_sabha" bson:"lok_sabha"`
    
    AdminType    string   `json:"administrative_type" bson:"administrative_type"`
    Headquarters string   `json:"headquarters" bson:"headquarters"`
    Region       string   `json:"region" bson:"region"`
    NearbyCities []string `json:"nearby_cities" bson:"nearby_cities"`
    
    Statistics struct {
        VillagesCount    int `json:"villages_count" bson:"villages_count"`
        PanchayatsCount  int `json:"panchayats_count" bson:"panchayats_count"`
        SchoolsCount     int `json:"schools_count" bson:"schools_count"`
        HospitalsCount   int `json:"hospitals_count" bson:"hospitals_count"`
        BanksCount       int `json:"banks_count" bson:"banks_count"`
    } `json:"statistics" bson:"statistics"`
    
    Languages        []string `json:"languages" bson:"languages"`
    PoliticalParties []string `json:"political_parties" bson:"political_parties"`
    
    Demographics struct {
        TotalPopulation    int     `json:"population_total" bson:"population_total"`
        MalePopulation     int     `json:"population_males" bson:"population_males"`
        FemalePopulation   int     `json:"population_females" bson:"population_females"`
        HousesCount        int     `json:"houses_count" bson:"houses_count"`
        LiteracyRate      float64 `json:"literacy_rate" bson:"literacy_rate"`
        SexRatio          float64 `json:"sex_ratio" bson:"sex_ratio"`
        WorkForcePercent  float64 `json:"workforce_percent" bson:"workforce_percent"`
    } `json:"demographics" bson:"demographics"`
    
    Infrastructure struct {
        RoadLength      float64 `json:"road_length" bson:"road_length"`
        RailwayStations int     `json:"railway_stations" bson:"railway_stations"`
        BusStations     int     `json:"bus_stations" bson:"bus_stations"`
        PowerSupply     string  `json:"power_supply" bson:"power_supply"`
        WaterSupply     string  `json:"water_supply" bson:"water_supply"`
    } `json:"infrastructure" bson:"infrastructure"`
    
    Pincodes      []Pincode      `json:"pincodes" bson:"pincodes"`
    Colleges      []College      `json:"colleges" bson:"colleges"`
    Rivers        []River        `json:"rivers" bson:"rivers"`
    TouristPlaces []TouristPlace `json:"tourist_places" bson:"tourist_places"`
    Companies     []Company      `json:"companies" bson:"companies"`
    Hospitals     []Hospital     `json:"hospitals" bson:"hospitals"`
    Banks         []Bank         `json:"banks" bson:"banks"`
    
    CreatedAt     string `json:"created_at" bson:"created_at"`
    UpdatedAt     string `json:"updated_at" bson:"updated_at"`
}

type Pincode struct {
    Code    string   `json:"code" bson:"code"`
    Area    string   `json:"area,omitempty" bson:"area,omitempty"`
    Type    string   `json:"type,omitempty" bson:"type,omitempty"`
    Villages []string `json:"villages,omitempty" bson:"villages,omitempty"`
}

type College struct {
    Name        string   `json:"name" bson:"name"`
    Type        string   `json:"type,omitempty" bson:"type,omitempty"`
    Address     string   `json:"address,omitempty" bson:"address,omitempty"`
    Courses     []string `json:"courses,omitempty" bson:"courses,omitempty"`
    Established string   `json:"established,omitempty" bson:"established,omitempty"`
    Website     string   `json:"website,omitempty" bson:"website,omitempty"`
}

// type River struct {
//     Name        string  `json:"name" bson:"name"`
//     Length      float64 `json:"length,omitempty" bson:"length,omitempty"`
//     Origin      string  `json:"origin,omitempty" bson:"origin,omitempty"`
//     Destination string  `json:"destination,omitempty" bson:"destination,omitempty"`
// }

type TouristPlace struct {
    Name        string   `json:"name" bson:"name"`
    Description string   `json:"description,omitempty" bson:"description,omitempty"`
    Type        string   `json:"type,omitempty" bson:"type,omitempty"`
    Distance    float64  `json:"distance,omitempty" bson:"distance,omitempty"`
    Activities  []string `json:"activities,omitempty" bson:"activities,omitempty"`
    BestTime    string   `json:"best_time,omitempty" bson:"best_time,omitempty"`
}

type Company struct {
    Name        string `json:"name" bson:"name"`
    Type        string `json:"type,omitempty" bson:"type,omitempty"`
    Industry    string `json:"industry,omitempty" bson:"industry,omitempty"`
    Employees   int    `json:"employees,omitempty" bson:"employees,omitempty"`
    Established string `json:"established,omitempty" bson:"established,omitempty"`
}

type Hospital struct {
    Name        string   `json:"name" bson:"name"`
    Type        string   `json:"type" bson:"type"`
    Address     string   `json:"address" bson:"address"`
    Beds        int      `json:"beds,omitempty" bson:"beds,omitempty"`
    Specialties []string `json:"specialties,omitempty" bson:"specialties,omitempty"`
    Emergency   bool     `json:"emergency" bson:"emergency"`
}

type Bank struct {
    Name     string `json:"name" bson:"name"`
    Branch   string `json:"branch" bson:"branch"`
    IFSC     string `json:"ifsc" bson:"ifsc"`
    Address  string `json:"address" bson:"address"`
    Type     string `json:"type" bson:"type"`
    Services []string `json:"services,omitempty" bson:"services,omitempty"`
}