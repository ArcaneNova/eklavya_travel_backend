package models

type CensusData struct {
    District                    string  `json:"district"`
    Subdistrict                string  `json:"subdistrict"`
    Village                    string  `json:"village"`
    TotalPopulation            int     `json:"total_population"`
    FemalePopulation          int     `json:"female_population"`
    TotalLiteracy             float64 `json:"total_literacy"`
    FemaleLiteracy            float64 `json:"female_literacy"`
    STPopulation              int     `json:"st_population"`
    WorkingPopulation         int     `json:"working_population"`
    GramPanchayat             string  `json:"gram_panchayat"`
    DistanceFromSubdistrict   float64 `json:"distance_from_subdistrict"`
    DistanceFromDistrict      float64 `json:"distance_from_district"`
    NearestTown               string  `json:"nearest_town"`
    NearestTownDistance       float64 `json:"nearest_town_distance"`
    
    // Educational Facilities
    GovtPrimarySchool         bool    `json:"govt_primary_school"`
    GovtDisabledSchool        bool    `json:"govt_disabled_school"`
    GovtEngineeringCollege    bool    `json:"govt_engineering_college"`
    GovtMedicalCollege        bool    `json:"govt_medical_college"`
    GovtPolytechnic           bool    `json:"govt_polytechnic"`
    GovtSecondarySchool       bool    `json:"govt_secondary_school"`
    GovtSeniorSecondary       bool    `json:"govt_senior_secondary"`
    NearestPrePrimary         float64 `json:"nearest_pre_primary"`
    NearestPolytechnic        float64 `json:"nearest_polytechnic"`
    NearestSecondary          float64 `json:"nearest_secondary"`
    
    // Infrastructure
    PrimaryHealthCenter       bool    `json:"primary_health_center"`
    CommunityHealthCenter     bool    `json:"community_health_center"`
    FamilyWelfareCenter      bool    `json:"family_welfare_center"`
    MaternityChildCenter     bool    `json:"maternity_child_center"`
    TBClinic                 bool    `json:"tb_clinic"`
    VeterinaryHospital       bool    `json:"veterinary_hospital"`
    MobileHealthClinic       bool    `json:"mobile_health_clinic"`
    MedicalShop              bool    `json:"medical_shop"`
    TreatedTapWater          bool    `json:"treated_tap_water"`
    UntreatedWater           bool    `json:"untreated_water"`
    CoveredWell              bool    `json:"covered_well"`
    UncoveredWell            bool    `json:"uncovered_well"`
    Handpump                 bool    `json:"handpump"`
    DrainageSystem           bool    `json:"drainage_system"`
    GarbageCollection        bool    `json:"garbage_collection"`
    DirectDrainDischarge     bool    `json:"direct_drain_discharge"`
    
    // Connectivity
    MobileCoverage           bool    `json:"mobile_coverage"`
    InternetCafe             bool    `json:"internet_cafe"`
    PrivateCourier           bool    `json:"private_courier"`
    BusService               bool    `json:"bus_service"`
    RailwayStation           bool    `json:"railway_station"`
    AnimalCart               bool    `json:"animal_cart"`
    NationalHighway          bool    `json:"national_highway"`
    StateHighway             bool    `json:"state_highway"`
    DistrictRoad             bool    `json:"district_road"`
    
    // Amenities
    ATM                      bool    `json:"atm"`
    CommercialBank           bool    `json:"commercial_bank"`
    CooperativeBank          bool    `json:"cooperative_bank"`
    PowerSupply              bool    `json:"power_supply"`
    Anganwadi                bool    `json:"anganwadi"`
    BirthDeathRegistration   bool    `json:"birth_death_registration"`
    Newspaper                bool    `json:"newspaper"`
    
    // Land Usage
    TotalArea               float64 `json:"total_area"`
    IrrigatedArea          float64 `json:"irrigated_area"`
}

type CensusStatistics struct {
    Data            CensusData `json:"census_data"`
    Statistics      struct {
        LiteracyRate        float64 `json:"literacy_rate"`
        WorkForceRate       float64 `json:"workforce_rate"`
        FemaleWorkForceRate float64 `json:"female_workforce_rate"`
        InfrastructureIndex float64 `json:"infrastructure_index"`
    } `json:"statistics"`
    Comparisons     struct {
        DistrictAvg    map[string]float64 `json:"district_averages"`
        StateAvg       map[string]float64 `json:"state_averages"`
    } `json:"comparisons"`
}