package handlers

import (
    "encoding/json"
    "log"
    "net/http"
    "village_site/config"
)

type CensusRequest struct {
    District    string `json:"district"`
    Subdistrict string `json:"subdistrict"`
    Village     string `json:"village"`
}

type CensusResponse struct {
    Basic struct {
        District    string `json:"district"`
        Subdistrict string `json:"subdistrict"`
        Village     string `json:"village"`
    } `json:"basic"`

    Demographics struct {
        TotalPopulation    float64 `json:"total_population"`
        FemalePopulation   float64 `json:"female_population"`
        TotalLiteracy      float64 `json:"total_literacy"`
        FemaleLiteracy     float64 `json:"female_literacy"`
        STPopulation       float64 `json:"st_population"`
        WorkingPopulation  float64 `json:"working_population"`
    } `json:"demographics"`

    Location struct {
        GramPanchayat            string  `json:"gram_panchayat"`
        DistanceFromSubdistrict  float64 `json:"distance_from_subdistrict"`
        DistanceFromDistrict     float64 `json:"distance_from_district"`
        NearestTown             string  `json:"nearest_town"`
        NearestTownDistance     float64 `json:"nearest_town_distance"`
    } `json:"location"`

    Education struct {
        GovtPrimarySchool       bool   `json:"govt_primary_school"`
        GovtDisabledSchool      bool   `json:"govt_disabled_school"`
        GovtEngineeringCollege  bool   `json:"govt_engineering_college"`
        GovtMedicalCollege      bool   `json:"govt_medical_college"`
        GovtPolytechnic         bool   `json:"govt_polytechnic"`
        GovtSecondarySchool     bool   `json:"govt_secondary_school"`
        GovtSeniorSecondary     bool   `json:"govt_senior_secondary"`
        NearestPrePrimary       string `json:"nearest_pre_primary"`
        NearestPolytechnic      string `json:"nearest_polytechnic"`
        NearestSecondary        string `json:"nearest_secondary"`
    } `json:"education"`

    Health struct {
        PrimaryHealthCenter    bool `json:"primary_health_center"`
        CommunityHealthCenter bool `json:"community_health_center"`
        FamilyWelfareCenter   bool `json:"family_welfare_center"`
        MaternityChildCenter  bool `json:"maternity_child_center"`
        TBClinic              bool `json:"tb_clinic"`
        VeterinaryHospital    bool `json:"veterinary_hospital"`
        MobileHealthClinic    bool `json:"mobile_health_clinic"`
        MedicalShop           bool `json:"medical_shop"`
    } `json:"health"`

    Infrastructure struct {
        TreatedTapWater      bool `json:"treated_tap_water"`
        UntreatedWater       bool `json:"untreated_water"`
        CoveredWell          bool `json:"covered_well"`
        UncoveredWell        bool `json:"uncovered_well"`
        Handpump             bool `json:"handpump"`
        DrainageSystem       bool `json:"drainage_system"`
        GarbageCollection    bool `json:"garbage_collection"`
        DirectDrainDischarge bool `json:"direct_drain_discharge"`
    } `json:"infrastructure"`

    Connectivity struct {
        MobileCoverage  bool `json:"mobile_coverage"`
        InternetCafe    bool `json:"internet_cafe"`
        PrivateCourier  bool `json:"private_courier"`
        BusService      bool `json:"bus_service"`
        RailwayStation  bool `json:"railway_station"`
        AnimalCart      bool `json:"animal_cart"`
    } `json:"connectivity"`

    Transport struct {
        NationalHighway bool `json:"national_highway"`
        StateHighway    bool `json:"state_highway"`
        DistrictRoad    bool `json:"district_road"`
    } `json:"transport"`

    Financial struct {
        ATM             bool `json:"atm"`
        CommercialBank  bool `json:"commercial_bank"`
        CooperativeBank bool `json:"cooperative_bank"`
    } `json:"financial"`

    OtherAmenities struct {
        PowerSupply              bool `json:"power_supply"`
        Anganwadi               bool `json:"anganwadi"`
        BirthDeathRegistration  bool `json:"birth_death_registration"`
        Newspaper               bool `json:"newspaper"`
    } `json:"other_amenities"`

    Area struct {
        TotalArea      float64 `json:"total_area"`
        IrrigatedArea  float64 `json:"irrigated_area"`
    } `json:"area"`
}

func GetCensusDetails(w http.ResponseWriter, r *http.Request) {
    var req CensusRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    log.Printf("Received census request for: District=%s, Subdistrict=%s, Village=%s",
        req.District, req.Subdistrict, req.Village)

    var response CensusResponse
    err := config.DB.QueryRow(`
        SELECT 
            district,
            subdistrict,
            village,
            COALESCE(NULLIF(trim(total_population::text), '')::float8, 0),
            COALESCE(NULLIF(trim(female_population::text), '')::float8, 0),
            COALESCE(NULLIF(trim(total_literacy::text), '')::float8, 0),
            COALESCE(NULLIF(trim(female_literacy::text), '')::float8, 0),
            COALESCE(NULLIF(trim(st_population::text), '')::float8, 0),
            COALESCE(NULLIF(trim(working_population::text), '')::float8, 0),
            
            COALESCE(gram_panchayat, ''),
            COALESCE(NULLIF(trim(distance_from_subdistrict::text), '')::float8, 0),
            COALESCE(NULLIF(trim(distance_from_district::text), '')::float8, 0),
            COALESCE(nearest_town, ''),
            COALESCE(NULLIF(trim(nearest_town_distance::text), '')::float8, 0),
            
            COALESCE(govt_primary_school, 0) = 1,
            COALESCE(govt_disabled_school, 0) = 1,
            COALESCE(govt_engineering_college, 0) = 1,
            COALESCE(govt_medical_college, 0) = 1,
            COALESCE(govt_polytechnic, 0) = 1,
            COALESCE(govt_secondary_school, 0) = 1,
            COALESCE(govt_senior_secondary, 0) = 1,
            COALESCE(nearest_pre_primary, ''),
            COALESCE(nearest_polytechnic, ''),
            COALESCE(nearest_secondary, ''),
            
            COALESCE(primary_health_center, 0) = 1,
            COALESCE(community_health_center, 0) = 1,
            COALESCE(family_welfare_center, 0) = 1,
            COALESCE(maternity_child_center, 0) = 1,
            COALESCE(tb_clinic, 0) = 1,
            COALESCE(veterinary_hospital, 0) = 1,
            COALESCE(mobile_health_clinic, 0) = 1,
            COALESCE(medical_shop, 0) = 1,
            
            COALESCE(treated_tap_water, 0) = 1,
            COALESCE(untreated_water, 0) = 1,
            COALESCE(covered_well, 0) = 1,
            COALESCE(uncovered_well, 0) = 1,
            COALESCE(handpump, 0) = 1,
            COALESCE(drainage_system, 0) = 1,
            COALESCE(garbage_collection, 0) = 1,
            COALESCE(direct_drain_discharge, 0) = 1,
            
            COALESCE(mobile_coverage, 0) = 1,
            COALESCE(internet_cafe, 0) = 1,
            COALESCE(private_courier, 0) = 1,
            COALESCE(bus_service, 0) = 1,
            COALESCE(railway_station, 0) = 1,
            COALESCE(animal_cart, 0) = 1,
            
            COALESCE(national_highway, 0) = 1,
            COALESCE(state_highway, 0) = 1,
            COALESCE(district_road, 0) = 1,
            
            COALESCE(atm, 0) = 1,
            COALESCE(commercial_bank, 0) = 1,
            COALESCE(cooperative_bank, 0) = 1,
            
            COALESCE(power_supply, 0) = 1,
            COALESCE(anganwadi, 0) = 1,
            COALESCE(birth_death_registration, 0) = 1,
            COALESCE(newspaper, 0) = 1,
            
            COALESCE(NULLIF(trim(total_area::text), '')::float8, 0),
            COALESCE(NULLIF(trim(irrigated_area::text), '')::float8, 0)
        FROM village_census
        WHERE LOWER(district) = LOWER($1)
        AND LOWER(subdistrict) = LOWER($2)
        AND LOWER(village) = LOWER($3)`,
        req.District, req.Subdistrict, req.Village).Scan(
            &response.Basic.District,
            &response.Basic.Subdistrict,
            &response.Basic.Village,
            &response.Demographics.TotalPopulation,
            &response.Demographics.FemalePopulation,
            &response.Demographics.TotalLiteracy,
            &response.Demographics.FemaleLiteracy,
            &response.Demographics.STPopulation,
            &response.Demographics.WorkingPopulation,
            
            &response.Location.GramPanchayat,
            &response.Location.DistanceFromSubdistrict,
            &response.Location.DistanceFromDistrict,
            &response.Location.NearestTown,
            &response.Location.NearestTownDistance,
            
            &response.Education.GovtPrimarySchool,
            &response.Education.GovtDisabledSchool,
            &response.Education.GovtEngineeringCollege,
            &response.Education.GovtMedicalCollege,
            &response.Education.GovtPolytechnic,
            &response.Education.GovtSecondarySchool,
            &response.Education.GovtSeniorSecondary,
            &response.Education.NearestPrePrimary,
            &response.Education.NearestPolytechnic,
            &response.Education.NearestSecondary,
            
            &response.Health.PrimaryHealthCenter,
            &response.Health.CommunityHealthCenter,
            &response.Health.FamilyWelfareCenter,
            &response.Health.MaternityChildCenter,
            &response.Health.TBClinic,
            &response.Health.VeterinaryHospital,
            &response.Health.MobileHealthClinic,
            &response.Health.MedicalShop,
            
            &response.Infrastructure.TreatedTapWater,
            &response.Infrastructure.UntreatedWater,
            &response.Infrastructure.CoveredWell,
            &response.Infrastructure.UncoveredWell,
            &response.Infrastructure.Handpump,
            &response.Infrastructure.DrainageSystem,
            &response.Infrastructure.GarbageCollection,
            &response.Infrastructure.DirectDrainDischarge,
            
            &response.Connectivity.MobileCoverage,
            &response.Connectivity.InternetCafe,
            &response.Connectivity.PrivateCourier,
            &response.Connectivity.BusService,
            &response.Connectivity.RailwayStation,
            &response.Connectivity.AnimalCart,
            
            &response.Transport.NationalHighway,
            &response.Transport.StateHighway,
            &response.Transport.DistrictRoad,
            
            &response.Financial.ATM,
            &response.Financial.CommercialBank,
            &response.Financial.CooperativeBank,
            
            &response.OtherAmenities.PowerSupply,
            &response.OtherAmenities.Anganwadi,
            &response.OtherAmenities.BirthDeathRegistration,
            &response.OtherAmenities.Newspaper,
            
            &response.Area.TotalArea,
            &response.Area.IrrigatedArea,
    )

    if err != nil {
        log.Printf("Error fetching census details: %v", err)
        http.Error(w, "Census data not found", http.StatusNotFound)
        return
    }

    // Set response headers
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Cache-Control", "public, max-age=300") // Cache for 5 minutes

    // Return response
    if err := json.NewEncoder(w).Encode(response); err != nil {
        log.Printf("Error encoding response: %v", err)
        http.Error(w, "Error encoding response", http.StatusInternalServerError)
        return
    }
}