package controllers

import (
	"bufio"
	"context"
	"drialbXauto/pkg/helpers"
	"drialbXauto/pkg/models"
	"drialbXauto/pkg/utils"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

var NewCar models.CarPost

const (
	clientID     = "214066364906-q6lct425sjn9f89n7b5ibs61cs1tp3ap.apps.googleusercontent.com"
	clientSecret = "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"
	redirectURI  = "https://developers.google.com/oauthplayground"
	refreshToken = "1//04-5wy7kbvT-5CgYIARAAGAQSNwF-L9IrNNzRLR59ibaXlJLS-GtTRrDvfHahBHdb0kwAW2Xl3Br2pTxVo9ITBNTlIBfNPEQnyD4"
)

var credentialsJSON = []byte(`
{
	"web": {
		"client_id": "` + clientID + `",
		"client_secret": "` + clientSecret + `",
		"redirect_uris": ["` + redirectURI + `"],
		"auth_uri": "https://accounts.google.com/o/oauth2/auth",
		"token_uri": "https://accounts.google.com/o/oauth2/token"
	}
}
`)

func GetCar(w http.ResponseWriter, r *http.Request) {
	// Retrieve user data from token
	tokenCookie, err := r.Cookie("token")
	var Data *models.User

	if err == nil && tokenCookie != nil {
		// Token found, verify it
		tokenString := tokenCookie.Value
		claims, err := helpers.VerifyToken(tokenString)
		if err == nil {
			// Token is valid, retrieve user information
			userID := claims.(jwt.MapClaims)["user_id"].(string)
			firstName := claims.(jwt.MapClaims)["first_name"].(string)
			lastName := claims.(jwt.MapClaims)["last_name"].(string)
			active := claims.(jwt.MapClaims)["active"].(string)

			fmt.Println("ACTIVEE+:", active)

			Data = &models.User{
				User_id:    userID,
				First_name: &firstName,
				Last_name:  &lastName,
				Active:     &active,
			}

		} else {
			// Invalid token, proceed with an empty user
			fmt.Println("Invalid token:", err)
			Data = &models.User{}
		}
	} else {
		// No token, proceed with an empty user
		Data = &models.User{}
	}

	// Import car makes, models, equipment, and fuel types
	err = ImportCarMakesAndModelsFromFile("../car_makes_and_models.txt")
	if err != nil {
		fmt.Println("Error:", err)
		http.Error(w, "Failed to import car makes and models", http.StatusInternalServerError)
		return
	}

	// Import boot space, if needed
	// trr := ImportBootSpaceFromFile("../car_bootSpace.txt")
	// if trr != nil {
	// 	fmt.Println("Error:", trr)
	// 	http.Error(w, "Failed to import car makes and models", http.StatusInternalServerError)
	// 	return
	// }

	// Import equipment
	brr := ImportEquipmentFromFile("../car_equipment.txt")
	if brr != nil {
		fmt.Println("Error:", brr)
		http.Error(w, "Failed to import car makes and models", http.StatusInternalServerError)
		return
	}

	// Import fuel types
	orr := ImportFuelTypesFromFile("../car_fuelType.txt")
	if orr != nil {
		fmt.Println("Error:", orr)
		http.Error(w, "Failed to import car makes and models", http.StatusInternalServerError)
		return
	}

	// Get all cars
	newCars := models.GetAllCars()
	models.GetOnePhotoPerCar()

	// Add user data to the template data
	templateData := struct {
		User *models.User
		Cars []models.CarPost
	}{
		User: Data,
		Cars: newCars,
	}

	// Parse the template and execute it with the template data
	tmpl, err := template.ParseFiles("../web/html/allCars/cars.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, templateData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type CarD struct {
	CarD1 *models.CarPost
	CarD2 []models.CarPhoto
	User  *models.User
}

func GetCarById(w http.ResponseWriter, r *http.Request) {
	// Retrieve user data from token
	tokenCookie, err := r.Cookie("token")
	var Data *models.User

	if err == nil && tokenCookie != nil {
		// Token found, verify it
		tokenString := tokenCookie.Value
		claims, err := helpers.VerifyToken(tokenString)
		if err == nil {
			// Token is valid, retrieve user information
			userID := claims.(jwt.MapClaims)["user_id"].(string)
			firstName := claims.(jwt.MapClaims)["first_name"].(string)
			lastName := claims.(jwt.MapClaims)["last_name"].(string)
			active := claims.(jwt.MapClaims)["active"].(string)

			fmt.Println("ACTIVEE+:", active)

			Data = &models.User{
				User_id:    userID,
				First_name: &firstName,
				Last_name:  &lastName,
				Active:     &active,
			}

		} else {
			// Invalid token, proceed with an empty user
			fmt.Println("Invalid token:", err)
			Data = &models.User{}
		}
	} else {
		// No token, proceed with an empty user
		Data = &models.User{}
	}

	vars := mux.Vars(r)
	carID := vars["carID"]
	fmt.Println("carID:", carID)

	ID, err := strconv.ParseInt(carID, 0, 0)
	if err != nil {
		fmt.Println("error while parsing")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid car ID"))
		return
	}

	carDetails, _ := models.GetCarById(ID)
	carDetailss, _ := models.GetPhotosByCarID(ID)

	tmpl, err := template.ParseFiles("../web/html/carByID.html")
	fmt.Println(carDetails.Phone)

	carDet := CarD{
		CarD1: carDetails,
		CarD2: carDetailss,
		User:  Data,
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, carDet)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func MyCars(w http.ResponseWriter, r *http.Request) {
	tokenCookie, err := r.Cookie("token")
	if err != nil || tokenCookie == nil {
		// Token not found in the cookies
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Println("error: Unauthorized")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	tokenString := tokenCookie.Value
	claims, err := helpers.VerifyToken(tokenString)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}
	userID := claims.(jwt.MapClaims)["user_id"].(string)
	// Fetch all cars with the user ID
	cars, err := models.GetCarsByUserID(userID)
	if err != nil {
		http.Error(w, "Failed to fetch cars", http.StatusInternalServerError)
		return
	}

	// Convert cars to JSON
	// jsonData, err := json.Marshal(cars)
	// if err != nil {
	// 	http.Error(w, "Failed to convert cars to JSON", http.StatusInternalServerError)
	// 	return
	// }

	tmpl, err := template.ParseFiles("../web/html/myCars.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, cars)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func CreateCar(w http.ResponseWriter, r *http.Request) {
	// Retrieve the list of makes from the database
	makes, err := models.GetAllMakes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Retrieve the list of equipment from the database
	equipment, err := models.GetAllEquipment()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Retrieve the list of fuelTypes from the database
	fuelType, err := models.GetAllFuelTypes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Retrieve the list of fuelTypes from the database
	bootSpace, err := models.GetALLBootSpace()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Pass the makes to the template for rendering the make dropdown options
	data := struct {
		Makes     []models.Make
		Models    []models.Model
		Equipment []models.Equipment
		FuelType  []models.FuelType
		BootSpace []models.BootSpace
	}{
		Makes:     makes,
		Equipment: equipment,
		FuelType:  fuelType,
		BootSpace: bootSpace,
	}

	// Get the selected make ID from the form
	makeID := r.FormValue("makeID")
	if makeID != "" {
		makeIDInt, err := strconv.Atoi(makeID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Retrieve the models for the selected make
		allModels, err := models.GetModelsByMakeID(makeIDInt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Pass the models to the template for rendering the model dropdown options
		data.Models = allModels
	}

	tmpl, err := template.ParseFiles("../web/html/createcar.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getClient(ctx context.Context, config *oauth2.Config) *http.Client {
	token := &oauth2.Token{RefreshToken: refreshToken}
	tokenSource := config.TokenSource(ctx, token)

	client := oauth2.NewClient(ctx, tokenSource)

	return client
}

func uploadFile(service *drive.Service, filePath string, fileName string) (*drive.File, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to open file: %v", err)
	}
	defer file.Close()

	// Create a new file on Google Drive with the specified file name
	driveFile, err := service.Files.Create(&drive.File{Name: fileName}).Media(file).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to upload file: %v", err)
	}

	return driveFile, nil
}

func generatePublicURL(service *drive.Service, fileID string) (string, error) {
	// Create public access permission
	permission := &drive.Permission{
		Role: "reader",
		Type: "anyone",
	}
	_, err := service.Permissions.Create(fileID, permission).Do()
	if err != nil {
		return "", fmt.Errorf("unable to create permission: %v", err)
	}

	// Retrieve file information with public URLs
	driveFile, err := service.Files.Get(fileID).Fields("webContentLink").Do()
	if err != nil {
		return "", fmt.Errorf("unable to get file information: %v", err)
	}

	// Return the webContentLink URL
	return driveFile.WebContentLink, nil
}

func generateThumbnailLink(fileID string) (string, error) {
	return "https://drive.google.com/thumbnail?id=" + fileID, nil
}

func Create(w http.ResponseWriter, r *http.Request) {
	// Parse the form data from the request
	err := r.ParseMultipartForm(10 << 20) // 10 MB maximum file size
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get the values of the form fields
	makeIdStr := r.FormValue("makeID")
	modelIdStr := r.FormValue("modelID")
	fuelType := r.FormValue("fuelType")
	bootSpace := r.FormValue("bootSpace")
	gearBox := r.FormValue("gearBox")
	yearStr := r.FormValue("year")
	mileage := r.FormValue("mileage")
	color := r.FormValue("color")
	priceStr := r.FormValue("price")
	descriptionc := r.FormValue("description")

	// Convert makeID, modelID, modelYear, year, price values to integers
	makeID, err := strconv.Atoi(makeIdStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	modelID, err := strconv.Atoi(modelIdStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	price, err := strconv.Atoi(priceStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Retrieve the make and model names from the database
	carMake, err := models.GetMakeByID(makeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	model, err := models.GetModelByID(modelID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set the VehicleType based on the selected model's VehicleType
	vehicleType := model.VehicleType

	// Retrieve selected equipment values as a slice
	equipment := r.Form["equipment"]

	// Convert equipment values into a comma-separated string
	equipmentValues := strings.Join(equipment, ", ")

	tokenCookie, err := r.Cookie("token")
	if err != nil || tokenCookie == nil {
		// Token not found in the cookies
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Println("error: Unauthorized")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	tokenString := tokenCookie.Value
	claims, err := helpers.VerifyToken(tokenString)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	userID := claims.(jwt.MapClaims)["user_id"].(string)
	firstName := claims.(jwt.MapClaims)["first_name"].(string)
	lastName := claims.(jwt.MapClaims)["last_name"].(string)
	phoneNr := claims.(jwt.MapClaims)["phone"].(string)

	// Create CarPost instance
	CreateCar := &models.CarPost{
		MakeID:      makeID,
		ModelID:     modelID,
		MakeName:    carMake.MakeName,
		ModelName:   model.ModelName,
		VehicleType: vehicleType,
		FuelType:    fuelType,
		GearBox:     gearBox,
		Equipment:   equipmentValues,
		BootSpace:   bootSpace,
		Year:        year,
		Mileage:     mileage,
		Color:       color,
		Price:       float64(price),
		Description: descriptionc,
		UserID:      userID,
		FirstName:   firstName,
		LastName:    lastName,
		Phone:       phoneNr,
	}

	// Parse the form files
	files := r.MultipartForm.File["images"]

	// Print file names
	for i, fileHeader := range files {
		fmt.Printf("File %d: %s\n", i+1, fileHeader.Filename)
	}

	// Authenticate and create Google Drive service
	ctx := context.Background()
	config, err := google.ConfigFromJSON(credentialsJSON, drive.DriveFileScope)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating Google Drive config: %v", err), http.StatusInternalServerError)
		return
	}

	client := getClient(ctx, config)
	service, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating Google Drive service: %v", err), http.StatusInternalServerError)
		return
	}

	// Create a wait group to wait for all file uploads to finish
	var wg sync.WaitGroup
	wg.Add(len(files))

	// Channel to receive file upload errors
	errCh := make(chan error, len(files))

	// Channel to collect file upload results
	uploadResults := make(chan models.CarPhoto, len(files))

	// Upload each photo to Google Drive concurrently
	var carPhotos []models.CarPhoto
	for index, fileHeader := range files {
		go func(index int, fileHeader *multipart.FileHeader) {
			defer wg.Done()

			// Get the file
			file, err := fileHeader.Open()
			if err != nil {
				errCh <- fmt.Errorf("Error opening file: %v", err)
				return
			}
			defer file.Close()

			// Create a temporary file to store the photo with a unique name
			tempFile, err := os.CreateTemp("", fmt.Sprintf("%s_X%d_*.jpg", userID, index))
			if err != nil {
				errCh <- fmt.Errorf("Error creating temporary file: %v", err)
				return
			}
			defer tempFile.Close()

			// Copy the file content to the temporary file
			_, err = io.Copy(tempFile, file)
			if err != nil {
				errCh <- fmt.Errorf("Error copying file content: %v", err)
				return
			}
			// Upload the photo to Google Drive
			photoDriveFile, err := uploadFile(service, tempFile.Name(), fmt.Sprintf("%s_photo%d.jpg", userID, index))
			if err != nil {
				log.Printf("Error uploading file to Google Drive: %v", err)
				errCh <- err
				return
			}

			// Add the file ID to the list
			photo := models.CarPhoto{
				CarID:     0, // Replace 0 with the actual car ID
				PhotoFile: photoDriveFile.Id,
			}

			// Generate public URL for the photo
			publicURL, err := generatePublicURL(service, photoDriveFile.Id)
			if err != nil {
				log.Printf("Error generating public URL: %v", err)
				errCh <- err
				return
			}

			// Generate thumbnail link for the photo
			thumbnailURL, err := generateThumbnailLink(photoDriveFile.Id)
			if err != nil {
				log.Printf("Error generating thumbnail link: %v", err)
				errCh <- err
				return
			}

			// Update the PhotoFile field with the generated links
			photo.PhotoFile = publicURL
			photo.PhotoFile = thumbnailURL

			// Send the result through the channel
			uploadResults <- photo
		}(index, fileHeader)
	}

	// Close the channel when all file uploads are complete
	go func() {
		wg.Wait()
		close(uploadResults)
	}()

	// Collect results from the channel
	for photo := range uploadResults {
		carPhotos = append(carPhotos, photo)
	}

	// Check if there were any errors during file uploads
	select {
	case err := <-errCh:
		log.Printf("Error during file uploads: %v", err)
		http.Error(w, fmt.Sprintf("Error during file uploads: %v", err), http.StatusInternalServerError)
		return
	default:
		// No errors, continue processing
	}

	// Add the car photos to your CarPost
	CreateCar.Photos = carPhotos

	// Save the car information in the database
	utils.ParseBody(r, CreateCar)
	b := CreateCar.CreateCar()
	res, _ := json.Marshal(b)
	w.Write(res)
}

// deleteCar Func
func DeleteCar(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	carID := vars["carID"]
	ID, err := strconv.ParseInt(carID, 10, 64)
	if err != nil {
		fmt.Println("Error while parsing car ID:", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid car ID"))
		return
	}

	err = models.DeleteCar(ID)
	if err != nil {
		fmt.Println("Error while deleting car:", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to delete car"))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func GetModelsByMakeIDHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the makeID from the route parameters
	vars := mux.Vars(r)
	makeID, err := strconv.Atoi(vars["makeID"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Retrieve the models for the given makeID
	models, err := models.GetModelsByMakeID(makeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Serialize the models to JSON and send the response
	jsonResponse, err := json.Marshal(map[string]interface{}{"models": models})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}

func GetUpdateCar(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	carID := vars["carID"]
	ID, err := strconv.ParseInt(carID, 0, 0)
	if err != nil {
		fmt.Println("error while parsing")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid car ID"))
		return
	}

	// Retrieve the car details by ID
	carDetails, _ := models.GetCarById(ID)

	// Retrieve additional data (makes, models, equipment, fuel types, boot space)
	makes, _ := models.GetAllMakes()
	equipment, _ := models.GetAllEquipment()
	fuelType, _ := models.GetAllFuelTypes()
	bootSpace, _ := models.GetALLBootSpace()

	// Pass the data to the template
	data := struct {
		CarDetails models.CarPost // Assuming Car is the type of carDetails
		Makes      []models.Make
		Models     []models.Model
		Equipment  []models.Equipment
		FuelType   []models.FuelType
		BootSpace  []models.BootSpace
	}{
		CarDetails: *carDetails, // Assuming carDetails is not nil
		Makes:      makes,
		Equipment:  equipment,
		FuelType:   fuelType,
		BootSpace:  bootSpace,
	}

	// Parse the template
	tmpl, err := template.ParseFiles("../web/html/updateCar.html")
	if err != nil {
		fmt.Println("Error parsing template:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Execute the template with the combined data
	err = tmpl.Execute(w, data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func UpdateCar(w http.ResponseWriter, r *http.Request) {
	// Parse the form data from the request
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get the car ID from the URL path parameters
	vars := mux.Vars(r)
	carID := vars["carID"]
	ID, err := strconv.ParseInt(carID, 0, 0)
	if err != nil {
		http.Error(w, "Invalid car ID", http.StatusBadRequest)
		return
	}

	// Retrieve the car details from the database
	carDetails, _ := models.GetCarById(ID)
	if err != nil || carDetails == nil {
		http.Error(w, "Car not found", http.StatusNotFound)
		return
	}

	priceStr := r.FormValue("price")
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		http.Error(w, "Invalid price", http.StatusBadRequest)
		return
	}

	// Update the car details with the form values
	// carDetails.MakeID, _ = strconv.Atoi(r.FormValue("makeID"))
	// carDetails.ModelID, _ = strconv.Atoi(r.FormValue("modelID"))
	// carDetails.MakeName = r.FormValue("makeName")
	// carDetails.ModelName = r.FormValue("modelName")
	// carDetails.ModelYear, _ = strconv.Atoi(r.FormValue("modelYear"))
	carDetails.VehicleType = r.FormValue("vehcileType")
	carDetails.Year, _ = strconv.Atoi(r.FormValue("year"))
	// carDetails.Milage = r.FormValue("mileage")
	carDetails.Color = r.FormValue("color")
	carDetails.Price = price
	carDetails.Description = r.FormValue("description")

	// Save the updated car details to the database
	err = carDetails.Save()
	if err != nil {
		http.Error(w, "Failed to update car", http.StatusInternalServerError)
		return
	}

	// Redirect the user to the car listing page
	http.Redirect(w, r, "/car/", http.StatusFound)
}

func ImportCarMakesAndModelsFromFile(filename string) error {
	fmt.Println("function ImportCarMakesAndModelsFromFile running")

	// Open the file in read mode
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a scanner to read the file line by line
	scanner := bufio.NewScanner(file)

	var make *models.Make

	// Iterate over each line in the file
	for scanner.Scan() {
		line := scanner.Text()

		// Split the line into fields
		fields := strings.Split(line, ", ")

		// Check the number of fields
		if len(fields) != 2 && len(fields) != 4 {
			continue
		}

		// Parse the fields
		if len(fields) == 2 {
			makeID := 0
			makeName := ""

			if _, err := fmt.Sscanf(fields[1], "MakeID: %d", &makeID); err != nil {
				continue
			}

			if _, err := fmt.Sscanf(fields[0], "MakeName: %s", &makeName); err != nil {
				continue
			}

			// Create a new make
			make = &models.Make{
				MakeID:   makeID,
				MakeName: makeName,
			}

			// Save the make to the database
			if err := make.Save(); err != nil {
				return err
			}
		}

		if len(fields) == 4 {
			modelID := 0
			modelName := ""
			makeID := 0
			vehicleType := ""

			if _, err := fmt.Sscanf(fields[0], "ModelID: %d", &modelID); err != nil {
				continue
			}

			if _, err := fmt.Sscanf(fields[1], "ModelName: %s", &modelName); err != nil {
				continue
			}

			if _, err := fmt.Sscanf(fields[2], "MakeID: %d", &makeID); err != nil {
				continue
			}

			if _, err := fmt.Sscanf(fields[3], "VehicleType: %s", &vehicleType); err != nil {
				continue
			}

			// Create a new model
			model := models.Model{
				MakeID:      makeID,
				ModelID:     modelID,
				ModelName:   modelName,
				VehicleType: vehicleType,
			}

			// Save the model to the database
			if err := model.Save(); err != nil {
				return err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func ImportEquipmentFromFile(filename string) error {
	// Open the file in read mode
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a scanner to read the file line by line
	scanner := bufio.NewScanner(file)

	// Iterate over each line in the file
	for scanner.Scan() {
		line := scanner.Text()

		// Split the line into fields
		fields := strings.Split(line, ", ")

		// Check the number of fields
		if len(fields) != 2 {
			continue
		}

		// Parse the fields
		equipmentID := 0
		equipmentName := ""

		if _, err := fmt.Sscanf(fields[0], "EquipmentID: %d", &equipmentID); err != nil {
			continue
		}

		if _, err := fmt.Sscanf(fields[1], "EquipmentName: %s", &equipmentName); err != nil {
			continue
		}

		// Create a new equipment
		equipment := models.Equipment{
			EquipmentID:   equipmentID,
			EquipmentName: equipmentName,
		}

		// Save the equipment to the database
		if err := equipment.Save(); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func ImportFuelTypesFromFile(filename string) error {
	// Open the file in read mode
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a scanner to read the file line by line
	scanner := bufio.NewScanner(file)

	// Iterate over each line in the file
	for scanner.Scan() {
		line := scanner.Text()

		// Split the line into fields
		fields := strings.Split(line, ", ")

		// Check the number of fields
		if len(fields) != 2 {
			continue
		}

		// Parse the fields
		fuelTypeID := 0
		fuelTypeName := ""

		if _, err := fmt.Sscanf(fields[0], "FuelTypeID: %d", &fuelTypeID); err != nil {
			continue
		}

		if _, err := fmt.Sscanf(fields[1], "FuelTypeName: %s", &fuelTypeName); err != nil {
			continue
		}

		// Create a new fuel type
		fuelType := models.FuelType{
			FuelTypeID:   fuelTypeID,
			FuelTypeName: fuelTypeName,
		}

		// Save the fuel type to the database
		if err := fuelType.Save(); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	fmt.Println("FuelTypeFUNC runing", filename)

	return nil
}

// ImportBootSpaceFromFile imports boot space data from a text file and saves it to the database.
func ImportBootSpaceFromFile(filename string) error {
	// Open the file in read mode
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a scanner to read the file line by line
	scanner := bufio.NewScanner(file)

	// Iterate over each line in the file
	for scanner.Scan() {
		line := scanner.Text()

		// Split the line into fields
		fields := strings.Fields(line)

		// Check the number of fields
		if len(fields) < 2 {
			continue
		}

		// Parse the fields
		bootSpaceName := strings.Join(fields[1:], " ") // Join the remaining fields

		// Create a new boot space
		bootSpace := models.BootSpace{
			BootSpace: bootSpaceName,
		}

		// Save the boot space to the database
		if err := bootSpace.Save(); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
