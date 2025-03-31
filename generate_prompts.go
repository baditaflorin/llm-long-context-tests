package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/go-faker/faker/v4" // Still used for Name generation
)

// --- Configuration ---
const (
	NUM_ENTRIES          = 5000
	MIN_AGE              = 18
	MAX_AGE              = 90
	OUTPUT_DIR           = "prompts_with_data_api_cities_list_jobs" // Changed output dir name
	CITY_API_URL         = "https://random-city-api.vercel.app/api/random-city"
	NUM_CITIES_TO_FETCH  = 150
	TARGET_UNIQUE_CITIES = 100
	API_REQUEST_DELAY    = 100 * time.Millisecond
)

// --- Predefined Job Titles List ---
var predefinedJobTitles = []string{
	"Software Engineer", "Project Manager", "Data Scientist", "Product Manager", "Accountant",
	"Graphic Designer", "Marketing Manager", "Sales Representative", "Customer Service Representative",
	"Human Resources Manager", "Teacher", "Nurse", "Doctor", "Lawyer", "Chef", "Mechanic",
	"Electrician", "Plumber", "Consultant", "Analyst", "Administrator", "Receptionist",
	"Web Developer", "UX Designer", "System Administrator", "DevOps Engineer", "Business Analyst",
	"Financial Advisor", "Architect", "Civil Engineer", "Mechanical Engineer", "Artist", "Writer",
	"Editor", "Photographer", "Scientist", "Researcher", "Librarian", "Police Officer", "Firefighter",
}

// --- Data Structures ---
type PersonEntry struct {
	Name     string
	Age      int
	City     string
	JobTitle string
}

type CityAPIResponse struct {
	City    string `json:"city"`
	Country string `json:"country"`
}

type PromptConfig struct {
	Desc              string
	QueryCount        int
	Template          string
	QueryIndices      []int
	IsSequential      bool
	NonExistentName   string
	IsReverseLookup   bool
	IsCombinedRequest bool
	IsConfirmation    bool
	IsMultiCity       bool
	IsMultiJob        bool
	IsMultiAgeCity    bool
	IsMultiCount      bool
}

// --- Helper Structs for Faker (Name only) ---
type nameHelper struct {
	FirstName string `faker:"first_name"`
	LastName  string `faker:"last_name"`
}

// --- Function to Fetch Cities from API --- (Unchanged)
func fetchCitiesFromAPI(numToFetch int, targetUnique int) ([]string, error) {
	fmt.Printf("Fetching up to %d cities from API (aiming for %d unique)...\n", numToFetch, targetUnique)
	cities := []string{}
	seenCities := make(map[string]bool)
	client := &http.Client{Timeout: 10 * time.Second}

	for i := 0; i < numToFetch && len(seenCities) < targetUnique; i++ {
		resp, err := client.Get(CITY_API_URL)
		if err != nil {
			log.Printf("Warning: Error fetching city (attempt %d): %v\n", i+1, err)
			time.Sleep(API_REQUEST_DELAY * 2)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			log.Printf("Warning: API non-OK status (attempt %d): %s\n", i+1, resp.Status)
			resp.Body.Close()
			time.Sleep(API_REQUEST_DELAY * 2)
			continue
		}

		var apiResp CityAPIResponse
		err = json.NewDecoder(resp.Body).Decode(&apiResp)
		resp.Body.Close()
		if err != nil {
			log.Printf("Warning: Error decoding API response (attempt %d): %v\n", i+1, err)
			continue
		}

		if apiResp.City != "" && !seenCities[apiResp.City] {
			seenCities[apiResp.City] = true
			cities = append(cities, apiResp.City)
			fmt.Printf("Fetched unique city %d: %s\n", len(cities), apiResp.City)
		} else if apiResp.City == "" {
			log.Printf("Warning: API returned empty city name (attempt %d)\n", i+1)
		}

		time.Sleep(API_REQUEST_DELAY)
	}

	if len(cities) == 0 {
		return nil, fmt.Errorf("failed to fetch any valid cities after %d attempts", numToFetch)
	}
	fmt.Printf("Finished fetching cities. Got %d unique cities.\n", len(cities))
	return cities, nil
}

// --- Function to Generate Random Data (Using API Cities & Predefined Jobs) ---
func generateRandomData(numEntries int, availableCities []string) ([]PersonEntry, error) {
	if len(availableCities) == 0 {
		return nil, fmt.Errorf("cannot generate data without any available cities")
	}
	if len(predefinedJobTitles) == 0 {
		return nil, fmt.Errorf("predefined job titles list is empty")
	} // Added check

	fmt.Printf("Generating %d random unique person entries using API cities and predefined jobs...\n", numEntries)
	data := make([]PersonEntry, 0, numEntries)
	usedNames := make(map[string]bool)
	attempts := 0
	maxAttempts := numEntries * 5
	var nameH nameHelper // Only need name helper now

	for len(data) < numEntries && attempts < maxAttempts {
		attempts++

		// Generate name using faker helper struct
		errName := faker.FakeData(&nameH)
		if errName != nil {
			log.Printf("Warning: Error generating faker name data: %v. Skipping entry.", errName)
			continue
		}
		name := fmt.Sprintf("%s %s", nameH.FirstName, nameH.LastName)

		if !usedNames[name] {
			usedNames[name] = true
			age := rand.Intn(MAX_AGE-MIN_AGE+1) + MIN_AGE
			// Assign a random city from the fetched list
			city := availableCities[rand.Intn(len(availableCities))]
			// Assign a random job title from the predefined list
			jobTitle := predefinedJobTitles[rand.Intn(len(predefinedJobTitles))]

			data = append(data, PersonEntry{Name: name, Age: age, City: city, JobTitle: jobTitle})
		}
	}

	if len(data) < numEntries {
		log.Printf("Warning: Could only generate %d unique names after %d attempts.", len(data), attempts)
	}

	rand.Shuffle(len(data), func(i, j int) { data[i], data[j] = data[j], data[i] })
	fmt.Printf("Data generation complete (%d unique entries generated).\n", len(data))
	return data, nil
}

// --- Function to Format Data Block --- (Unchanged)
func formatDataBlock(data []PersonEntry) string { /* ... as before ... */
	var builder strings.Builder
	for i, entry := range data {
		builder.WriteString(fmt.Sprintf("Name: %s | Age: %d | City: %s | Job Title: %s", entry.Name, entry.Age, entry.City, entry.JobTitle))
		if i < len(data)-1 {
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

// --- Helper Functions for Random Sampling --- (Unchanged)
func randomSampleNames(names []string, k int) []string { /* ... as before ... */
	n := len(names)
	if k < 0 {
		k = 0
	}
	if k > n {
		k = n
	}
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}
	rand.Shuffle(n, func(i, j int) { indices[i], indices[j] = indices[j], indices[i] })
	sampledNames := make([]string, k)
	for i := 0; i < k; i++ {
		sampledNames[i] = names[indices[i]]
	}
	return sampledNames
}
func randomSampleEntries(entries []PersonEntry, k int) []PersonEntry { /* ... as before ... */
	n := len(entries)
	if k < 0 {
		k = 0
	}
	if k > n {
		k = n
	}
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}
	rand.Shuffle(n, func(i, j int) { indices[i], indices[j] = indices[j], indices[i] })
	sampledEntries := make([]PersonEntry, k)
	for i := 0; i < k; i++ {
		sampledEntries[i] = entries[indices[i]]
	}
	return sampledEntries
}

// --- Main Function ---
func main() {
	rand.Seed(time.Now().UnixNano())

	// --- Fetch Cities First ---
	fetchedCities, err := fetchCitiesFromAPI(NUM_CITIES_TO_FETCH, TARGET_UNIQUE_CITIES)
	if err != nil {
		log.Fatalf("Critical error fetching cities: %v. Exiting.", err)
	}
	if len(fetchedCities) == 0 {
		log.Fatal("No cities were fetched successfully. Exiting.")
	}

	// --- Generate Master Data Using Fetched Cities & Predefined Jobs ---
	masterData, err := generateRandomData(NUM_ENTRIES, fetchedCities)
	if err != nil {
		log.Fatalf("Critical error generating person data: %v. Exiting.", err)
	}
	if len(masterData) == 0 {
		log.Fatal("No person data was generated successfully. Exiting.")
	}

	dataBlockString := formatDataBlock(masterData)
	allNames := make([]string, len(masterData))
	for i, entry := range masterData {
		allNames[i] = entry.Name
	}

	// --- Define Prompt Configurations (Templates remain the same) ---
	// (Same PromptConfig slice definition as the previous multi-attribute version)
	promptConfigs := []PromptConfig{
		{Desc: "01_standard_retrieval_10", QueryCount: 10, Template: `Here is the list:\n{{.DataBlock}}\n\nFrom the list above, what are the ages for:\n{{.QueryItemsFormatted}}`},
		{Desc: "02_different_phrasing_10", QueryCount: 10, Template: `See the following data:\n{{.DataBlock}}\n\nUsing only this data, find the ages associated with these names: {{.QueryItemsFormattedInline}}.`},
		{Desc: "03_fewer_items_5", QueryCount: 5, Template: `Data:\n{{.DataBlock}}\n\nProvide the ages for:\n{{.QueryItemsFormatted}}`},
		{Desc: "04_more_items_15", QueryCount: 15, Template: `List:\n{{.DataBlock}}\n\nPlease list the ages for the following 15 people:\n{{.QueryItemsFormatted}}`},
		{Desc: "05_start_end_focus_2", QueryIndices: []int{1, len(masterData) - 2}, Template: `Dataset:\n{{.DataBlock}}\n\nWhat is the age of {{.QueryName1}} and the age of {{.QueryName2}} from this dataset?`},
		{Desc: "06_reverse_lookup_name", QueryCount: 2, IsReverseLookup: true, Template: `Names and Ages:\n{{.DataBlock}}\n\nBased on the list, which person has age {{.QueryAge1}}? And who has age {{.QueryAge2}}? (If ages are not unique, list all names found)`},
		{Desc: "07_combined_request", QueryCount: 3, IsCombinedRequest: true, Template: `Reference Data:\n{{.DataBlock}}\n\nFind the age for {{.QueryName1}}. Also, find the age for {{.QueryName2}}. Finally, find the name associated with age {{.QueryAge3}}.`},
		{Desc: "08_sequential_names_5", IsSequential: true, Template: `Data Log:\n{{.DataBlock}}\n\nWhat are the ages for {{.QueryName1}}, {{.QueryName2}}, {{.QueryName3}}, {{.QueryName4}}, and {{.QueryName5}}?`},
		{Desc: "09_widely_spaced_names_10", QueryCount: 10, Template: `People List:\n{{.DataBlock}}\n\nExtract ages for: {{.QueryItemsFormattedInline}}.`},
		{Desc: "10_retrieval_confirmation", QueryCount: 8, IsConfirmation: true, NonExistentName: "Slartibartfast", Template: `Master List:\n{{.DataBlock}}\n\nProvide ages for {{.QueryItemsFormattedInline}}. Also, confirm if '{{.NonExistentName}}' is present in this list.`},
		// Multi-Attribute Prompts
		{Desc: "11_filter_city_get_name_job", IsMultiCity: true, Template: `List Detail:\n{{.DataBlock}}\n\nList the names and job titles of all people in the list who live in the city '{{.TargetCity}}'.`},
		{Desc: "12_filter_job_get_name_age", IsMultiJob: true, Template: `Employee Data:\n{{.DataBlock}}\n\nFind the names and ages of everyone listed with the job title '{{.TargetJobTitle}}'.`},
		{Desc: "13_filter_age_city_get_name", IsMultiAgeCity: true, Template: `Resident Information:\n{{.DataBlock}}\n\nWho in the list is between {{.MinAge}} and {{.MaxAge}} years old AND lives in '{{.TargetCity}}'? List their full names.`},
		{Desc: "14_count_job_city", IsMultiCount: true, Template: `Census Data:\n{{.DataBlock}}\n\nHow many people in the list have the job title '{{.TargetJobTitle}}' AND live in the city '{{.TargetCity}}'? Provide only the count.`},
		{Desc: "15_filter_job_retrieve_all", IsMultiJob: true, Template: `Personnel Files:\n{{.DataBlock}}\n\nProvide all available details (Name, Age, City, Job Title) for everyone whose job title is '{{.TargetJobTitle}}'.`},
	}

	// --- Create Directory and Files ---
	err = os.MkdirAll(OUTPUT_DIR, 0755)
	if err != nil {
		log.Fatalf("Error creating directory %s: %v", OUTPUT_DIR, err)
	}
	fmt.Printf("\nGenerating complete prompt files using API cities & list jobs in directory: '%s'\n", OUTPUT_DIR)

	generatedCount := 0
	for _, config := range promptConfigs {
		// (Logic for populating templateData and writing files remains the same)
		// --- Start File Writing Logic ---
		filename := fmt.Sprintf("prompt_%s.txt", config.Desc)
		filepath := filepath.Join(OUTPUT_DIR, filename)
		templateData := map[string]interface{}{"DataBlock": dataBlockString}
		canGenerate := true

		// Populate templateData based on config type
		// (This large block is identical to the previous version - it populates based on flags like IsMultiCity etc.)
		// START POPULATE BLOCK
		if config.QueryCount > 0 {
			minRequiredData := config.QueryCount
			if config.IsReverseLookup {
				minRequiredData = 2
			}
			if config.IsCombinedRequest {
				minRequiredData = 3
			}
			if len(masterData) < minRequiredData {
				log.Printf("Warning: Not enough data (%d) for query type in %s (needs %d). Skipping.", len(masterData), config.Desc, minRequiredData)
				canGenerate = false
			} else {
				selectedNames := randomSampleNames(allNames, config.QueryCount)
				templateData["QueryItemsFormatted"] = "- " + strings.Join(selectedNames, "\n- ")
				templateData["QueryItemsFormattedInline"] = strings.Join(selectedNames, ", ")
				if config.IsReverseLookup {
					selectedEntries := randomSampleEntries(masterData, 2)
					templateData["QueryAge1"] = selectedEntries[0].Age
					templateData["QueryAge2"] = selectedEntries[1].Age
				} else if config.IsCombinedRequest {
					selectedEntries := randomSampleEntries(masterData, 3)
					templateData["QueryName1"] = selectedEntries[0].Name
					templateData["QueryName2"] = selectedEntries[1].Name
					templateData["QueryAge3"] = selectedEntries[2].Age
				} else if config.IsConfirmation {
					if len(allNames) < config.QueryCount {
						selectedNames = randomSampleNames(allNames, len(allNames))
					}
					templateData["QueryItemsFormattedInline"] = strings.Join(selectedNames, ", ")
					templateData["NonExistentName"] = config.NonExistentName
				}
			}
		} else if len(config.QueryIndices) > 0 {
			idx1 := config.QueryIndices[0]
			idx2 := config.QueryIndices[1]
			realIdx1 := idx1
			realIdx2 := idx2
			if realIdx1 >= len(masterData) {
				realIdx1 = len(masterData) - 1
			}
			if realIdx2 >= len(masterData) {
				realIdx2 = len(masterData) - 1
			}
			if realIdx1 < 0 || realIdx2 < 0 {
				log.Printf("Warning: Invalid query indices for %s. Skipping.", config.Desc)
				canGenerate = false
			} else {
				templateData["QueryName1"] = masterData[realIdx1].Name
				templateData["QueryName2"] = masterData[realIdx2].Name
			}
		} else if config.IsSequential {
			if len(masterData) < 5 {
				log.Printf("Warning: Not enough data (%d) for sequential query in %s. Skipping.", len(masterData), config.Desc)
				canGenerate = false
			} else {
				startIndex := rand.Intn(len(masterData) - 4)
				for i := 0; i < 5; i++ {
					templateData[fmt.Sprintf("QueryName%d", i+1)] = masterData[startIndex+i].Name
				}
			}
		} else if config.IsMultiCity {
			if len(masterData) == 0 {
				canGenerate = false
			} else {
				templateData["TargetCity"] = masterData[rand.Intn(len(masterData))].City
			}
		} else if config.IsMultiJob {
			if len(masterData) == 0 {
				canGenerate = false
			} else {
				templateData["TargetJobTitle"] = masterData[rand.Intn(len(masterData))].JobTitle
			}
		} else if config.IsMultiAgeCity {
			if len(masterData) == 0 {
				canGenerate = false
			} else {
				templateData["TargetCity"] = masterData[rand.Intn(len(masterData))].City
				midAge := masterData[rand.Intn(len(masterData))].Age
				minAgeQuery := midAge - 5
				maxAgeQuery := midAge + 5
				if minAgeQuery < MIN_AGE {
					minAgeQuery = MIN_AGE
				}
				if maxAgeQuery > MAX_AGE {
					maxAgeQuery = MAX_AGE
				}
				if minAgeQuery > maxAgeQuery {
					minAgeQuery = maxAgeQuery
				}
				templateData["MinAge"] = strconv.Itoa(minAgeQuery)
				templateData["MaxAge"] = strconv.Itoa(maxAgeQuery)
			}
		} else if config.IsMultiCount {
			if len(masterData) == 0 {
				canGenerate = false
			} else {
				templateData["TargetJobTitle"] = masterData[rand.Intn(len(masterData))].JobTitle
				templateData["TargetCity"] = masterData[rand.Intn(len(masterData))].City
			}
		}
		// END POPULATE BLOCK

		if !canGenerate {
			continue
		}

		tmpl, err := template.New(config.Desc).Parse(config.Template)
		if err != nil {
			log.Printf("Error parsing template for %s: %v", config.Desc, err)
			continue
		}
		var buf bytes.Buffer
		err = tmpl.Execute(&buf, templateData)
		if err != nil {
			log.Printf("Error executing template for %s: %v", config.Desc, err)
			continue
		}
		err = os.WriteFile(filepath, buf.Bytes(), 0644)
		if err != nil {
			log.Printf("Error writing file %s: %v", filepath, err)
		} else {
			fmt.Printf("Successfully created: %s\n", filepath)
			generatedCount++
		}
		// --- End File Writing Logic ---
	}

	fmt.Printf("\nScript finished. Generated %d prompt files.\n", generatedCount)
	fmt.Printf("The generated files in '%s' contain the full list and are ready to be copied and pasted.\n", OUTPUT_DIR)
}
