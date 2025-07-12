package app

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/OpenConnectOUSL/backend-api-v1/internal/utils"
	"github.com/OpenConnectOUSL/backend-api-v1/internal/validator"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

// envelope is a type alias for map[string]any used for JSON responses
type Envelope map[string]any

// ReadIDParam reads and parses a UUID from the URL parameters
func (app *Application) ReadIDParam(r *http.Request) (uuid.UUID, error) {
	params := httprouter.ParamsFromContext(r.Context())
	idStr := params.ByName("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, errors.New("invalid id parameter")
	}
	return id, nil
}

// WriteJSON writes a JSON response with the given status code and data
func (app *Application) WriteJSON(w http.ResponseWriter, status int, data Envelope, headers http.Header) error {
	js, err := json.Marshal(data)
	if err != nil {
		return err
	}
	js = append(js, '\n')

	for key, value := range headers {
		w.Header()[key] = value
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Status", http.StatusText(status))
	w.WriteHeader(status)

	_, err = w.Write(js)
	return err
}

// ReadJSON reads and decodes JSON from the request body
func (app *Application) ReadJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	maxBytes := 1_048_576 * 10 // 10MB
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")

		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)

		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key %s", fieldName)

		case errors.As(err, &maxBytesError):
			return fmt.Errorf("body must not be larger than %d bytes", maxBytesError.Limit)

		case errors.As(err, &invalidUnmarshalError):
			panic(err)

		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must only contain a single JSON value")
	}
	return nil
}

// IsPDF checks if the provided data is a PDF file
func (app *Application) IsPDF(data []byte) bool {
	return bytes.HasPrefix(data, []byte("%PDF-"))
}

// ProcessAndSavePDF processes a base64 encoded PDF and saves it to disk
func (app *Application) ProcessAndSavePDF(inputBase64 string, w http.ResponseWriter, r *http.Request) (string, error) {
	pdfData, err := base64.StdEncoding.DecodeString(inputBase64)
	if err != nil {
		app.BadRequestResponse(w, r, err)
		return "no key", err
	}

	if !app.IsPDF(pdfData) {
		app.BadRequestResponse(w, r, fmt.Errorf("invalid pdf file"))
		return "no key", err
	}

	const maxPDFSize = 5 * 1024 * 1024 // 5MB
	if len(pdfData) > maxPDFSize {
		app.BadRequestResponse(w, r, fmt.Errorf("pdf file size must be less than 5MB"))
		return "no key", err
	}

	uploadsDir := "../../uploads"
	if _, err := os.Stat(uploadsDir); os.IsNotExist(err) {
		err = os.MkdirAll(uploadsDir, 0755)
		if err != nil {
			app.ServerErrorResponse(w, r, err)
			return "no key", err
		}
	}

	unique := utils.GenerateUUID()
	filenameWithId := unique + ".pdf"
	pdfPath := filepath.Join(uploadsDir, filenameWithId)

	err = os.WriteFile(pdfPath, pdfData, 0644)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return "no key", err
	}
	return unique, nil
}

// ProcessAndSaveAvatar processes a base64 encoded image and saves it as an avatar
func (app *Application) ProcessAndSaveAvatar(inputBase64 string, w http.ResponseWriter, r *http.Request) (string, error) {
	if inputBase64 == "" {
		return "no key", nil
	}

	imgData, err := base64.StdEncoding.DecodeString(inputBase64)
	if err != nil {
		app.BadRequestResponse(w, r, err)
		return "", fmt.Errorf("invalid image file: %w", err)
	}

	if len(imgData) < 8 {
		app.BadRequestResponse(w, r, fmt.Errorf("invalid image file"))
		return "", err
	}

	isJPEG := bytes.HasPrefix(imgData, []byte{0xFF, 0xD8, 0xFF})
	isPNG := bytes.HasPrefix(imgData, []byte{0x89, 0x50, 0x4E, 0x47})
	isGIF := bytes.HasPrefix(imgData, []byte{0x47, 0x49, 0x46, 0x38})

	if !isJPEG && !isPNG && !isGIF {
		app.BadRequestResponse(w, r, fmt.Errorf("invalid image file"))
		return "", fmt.Errorf("unsupported image format: %w", err)
	}

	const maxImageSize = 5 * 1024 * 1024 // 5MB
	if len(imgData) > maxImageSize {
		app.BadRequestResponse(w, r, fmt.Errorf("image file size must be less than 5MB"))
		return "", fmt.Errorf("image too large: %w", err)
	}

	uploadsDir := "../../uploads/avatars"
	if _, err := os.Stat(uploadsDir); os.IsNotExist(err) {
		err = os.MkdirAll(uploadsDir, 0755)
		if err != nil {
			app.ServerErrorResponse(w, r, err)
			return "", err
		}
	}

	unique := utils.GenerateUUID()

	var extension string
	if isJPEG {
		extension = ".jpg"
	} else if isPNG {
		extension = ".png"
	} else if isGIF {
		extension = ".gif"
	}

	filenameWithId := unique + extension
	imgPath := filepath.Join(uploadsDir, filenameWithId)

	err = os.WriteFile(imgPath, imgData, 0644)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return "", err
	}

	return unique, nil
}

func (app *Application) ReadString(qs url.Values, key string, defaultValue string) string {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}
	return s
}

func (app *Application) ReadCSV(qs url.Values, key string, defaultValue []string) []string {
	csv := qs.Get(key)
	if csv == "" {
		return defaultValue
	}
	return strings.Split(csv, ",")
}

func (app *Application) ReadInt(qs url.Values, key string, defaultValue int, v *validator.Validator) int {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		v.AddError(key, "must be an integer value")
		return defaultValue
	}

	return i
}
