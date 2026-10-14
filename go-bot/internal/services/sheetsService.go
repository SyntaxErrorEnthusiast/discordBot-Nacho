// internal/services/sheetsService.go
// Business logic for fetching and rasterising public Google Sheets.

package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/image/draw"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// ErrSheetNotPublic is returned when Google redirects to a login page,
// indicating the sheet is private or inaccessible without authentication.
var ErrSheetNotPublic = errors.New("sheet is not public or could not be accessed")

// parseSheetURL extracts the spreadsheet ID and sheet gid from a Google Sheets URL.
// Returns an error if the URL does not match the expected Google Sheets format.
func parseSheetURL(rawSheetURL string) (spreadsheetID string, sheetGID int64, parseError error) {
	spreadsheetIDPattern := regexp.MustCompile(`/spreadsheets/d/([a-zA-Z0-9_-]+)`)
	sheetGIDPattern := regexp.MustCompile(`[#&?]gid=(\d+)`)

	spreadsheetIDMatch := spreadsheetIDPattern.FindStringSubmatch(rawSheetURL)
	if spreadsheetIDMatch == nil {
		return "", 0, fmt.Errorf("could not parse a spreadsheet ID from the provided URL")
	}
	spreadsheetID = spreadsheetIDMatch[1]

	sheetGIDMatch := sheetGIDPattern.FindStringSubmatch(rawSheetURL)
	if sheetGIDMatch != nil {
		sheetGID, _ = strconv.ParseInt(sheetGIDMatch[1], 10, 64)
	}

	return spreadsheetID, sheetGID, nil
}

// columnIndexToLetter converts a zero-indexed column number to its A1-notation letter(s).
// For example: 0 → "A", 25 → "Z", 26 → "AA".
func columnIndexToLetter(columnIndex int) string {
	columnNumber := columnIndex + 1
	columnLetters := ""
	for columnNumber > 0 {
		remainder := (columnNumber - 1) % 26
		columnLetters = string(rune('A'+remainder)) + columnLetters
		columnNumber = (columnNumber - 1) / 26
	}
	return columnLetters
}

// getUsedCellRange returns the A1 notation bounding box of non-empty cells in the sheet,
// for example "A1:F20". Only cells with non-whitespace content are considered.
func getUsedCellRange(
	requestContext context.Context,
	spreadsheetID string,
	sheetGID int64,
	sheetsAPIKey string,
) (string, error) {
	sheetsService, serviceCreationError := sheets.NewService(requestContext, option.WithAPIKey(sheetsAPIKey))
	if serviceCreationError != nil {
		return "", fmt.Errorf("creating Google Sheets API client: %w", serviceCreationError)
	}

	spreadsheetMetadata, metadataFetchError := sheetsService.Spreadsheets.Get(spreadsheetID).
		Fields("sheets.properties(sheetId,title)").
		Context(requestContext).
		Do()
	if metadataFetchError != nil {
		return "", fmt.Errorf("fetching spreadsheet metadata: %w", metadataFetchError)
	}

	var sheetTitle string
	for _, sheetEntry := range spreadsheetMetadata.Sheets {
		if sheetEntry.Properties != nil && sheetEntry.Properties.SheetId == sheetGID {
			sheetTitle = sheetEntry.Properties.Title
			break
		}
	}
	if sheetTitle == "" {
		return "", fmt.Errorf("no sheet found with gid %d", sheetGID)
	}

	// Escape single quotes in the sheet title for safe use in A1 notation range strings.
	escapedSheetTitle := strings.ReplaceAll(sheetTitle, "'", "''")
	fullSheetRange := fmt.Sprintf("'%s'", escapedSheetTitle)

	cellValuesResponse, valuesFetchError := sheetsService.Spreadsheets.Values.Get(spreadsheetID, fullSheetRange).
		ValueRenderOption("FORMATTED_VALUE").
		Context(requestContext).
		Do()
	if valuesFetchError != nil {
		return "", fmt.Errorf("fetching sheet cell values: %w", valuesFetchError)
	}

	topRow, bottomRow, leftColumn, rightColumn := -1, -1, -1, -1

	for rowIndex, rowValues := range cellValuesResponse.Values {
		for columnIndex, cellValue := range rowValues {
			if cellValue != nil && strings.TrimSpace(fmt.Sprintf("%v", cellValue)) != "" {
				if topRow < 0 || rowIndex < topRow {
					topRow = rowIndex
				}
				if rowIndex > bottomRow {
					bottomRow = rowIndex
				}
				if leftColumn < 0 || columnIndex < leftColumn {
					leftColumn = columnIndex
				}
				if columnIndex > rightColumn {
					rightColumn = columnIndex
				}
			}
		}
	}

	if bottomRow < 0 {
		return "", fmt.Errorf("the sheet appears to be empty")
	}

	return fmt.Sprintf(
		"%s%d:%s%d",
		columnIndexToLetter(leftColumn), topRow+1,
		columnIndexToLetter(rightColumn), bottomRow+1,
	), nil
}

// exportSheetAsPDF requests a PDF export of a specific cell range from a public Google Sheet.
// Returns ErrSheetNotPublic if Google redirects to authentication.
func exportSheetAsPDF(spreadsheetID string, sheetGID int64, cellRange string) ([]byte, error) {
	exportQueryParams := url.Values{
		"format":        {"pdf"},
		"gid":           {strconv.FormatInt(sheetGID, 10)},
		"range":         {cellRange},
		"size":          {"A4"},
		"portrait":      {"false"},
		"scale":         {"4"},
		"gridlines":     {"false"},
		"printtitle":    {"false"},
		"sheetnames":    {"false"},
		"pagenum":       {"UNDEFINED"},
		"fzr":           {"false"},
		"top_margin":    {"0"},
		"bottom_margin": {"0"},
		"left_margin":   {"0"},
		"right_margin":  {"0"},
	}

	exportURL := fmt.Sprintf(
		"https://docs.google.com/spreadsheets/d/%s/export?%s",
		spreadsheetID,
		exportQueryParams.Encode(),
	)

	httpResponse, requestError := http.Get(exportURL) //nolint:noctx
	if requestError != nil {
		return nil, fmt.Errorf("PDF export HTTP request failed: %w", requestError)
	}
	defer httpResponse.Body.Close()

	// Google redirects to accounts.google.com when the sheet requires authentication.
	if strings.Contains(httpResponse.Request.URL.Host, "accounts.google.com") ||
		httpResponse.StatusCode == http.StatusUnauthorized ||
		httpResponse.StatusCode == http.StatusForbidden {
		return nil, ErrSheetNotPublic
	}

	if httpResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PDF export request returned unexpected status %d", httpResponse.StatusCode)
	}

	responseBuffer := new(bytes.Buffer)
	_, readError := responseBuffer.ReadFrom(httpResponse.Body)
	return responseBuffer.Bytes(), readError
}

// renderPDFPageToTrimmedPNG rasterises the first page of a PDF to a white-trimmed PNG image.
// Uses pdftoppm (from poppler-utils) as a subprocess for reliable rendering.
func renderPDFPageToTrimmedPNG(pdfBytes []byte) ([]byte, error) {
	temporaryDirectory, tempDirError := os.MkdirTemp("", "sheets-pdf-*")
	if tempDirError != nil {
		return nil, fmt.Errorf("creating temporary directory: %w", tempDirError)
	}
	defer os.RemoveAll(temporaryDirectory)

	pdfFilePath := filepath.Join(temporaryDirectory, "sheet.pdf")
	if writeError := os.WriteFile(pdfFilePath, pdfBytes, 0600); writeError != nil {
		return nil, fmt.Errorf("writing PDF to temporary file: %w", writeError)
	}

	outputFilePrefix := filepath.Join(temporaryDirectory, "page")

	// -r 288 = 288 DPI (equivalent to 4x scale at 72 DPI base), -l 1 = first page only
	pdftoppmCommand := exec.Command("pdftoppm", "-r", "288", "-l", "1", "-png", pdfFilePath, outputFilePrefix)
	if commandOutput, commandError := pdftoppmCommand.CombinedOutput(); commandError != nil {
		return nil, fmt.Errorf("pdftoppm rendering failed: %w\n%s", commandError, commandOutput)
	}

	renderedPageFiles, globError := filepath.Glob(filepath.Join(temporaryDirectory, "page-*.png"))
	if globError != nil || len(renderedPageFiles) == 0 {
		return nil, fmt.Errorf("pdftoppm produced no output pages")
	}

	renderedPageBytes, readError := os.ReadFile(renderedPageFiles[0])
	if readError != nil {
		return nil, fmt.Errorf("reading rendered page file: %w", readError)
	}

	sourceImage, _, decodeError := image.Decode(bytes.NewReader(renderedPageBytes))
	if decodeError != nil {
		return nil, fmt.Errorf("decoding rendered page image: %w", decodeError)
	}

	trimmedImage := trimWhiteBorder(sourceImage)

	const paddingPixels = 12
	trimmedBounds := trimmedImage.Bounds()
	paddedImageWidth := trimmedBounds.Dx() + paddingPixels*2
	paddedImageHeight := trimmedBounds.Dy() + paddingPixels*2

	paddedImage := image.NewRGBA(image.Rect(0, 0, paddedImageWidth, paddedImageHeight))
	draw.Draw(paddedImage, paddedImage.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	draw.Draw(
		paddedImage,
		image.Rect(paddingPixels, paddingPixels, paddingPixels+trimmedBounds.Dx(), paddingPixels+trimmedBounds.Dy()),
		trimmedImage,
		trimmedBounds.Min,
		draw.Src,
	)

	var encodedPNGBuffer bytes.Buffer
	if encodeError := png.Encode(&encodedPNGBuffer, paddedImage); encodeError != nil {
		return nil, fmt.Errorf("encoding final PNG: %w", encodeError)
	}

	return encodedPNGBuffer.Bytes(), nil
}

// trimWhiteBorder crops the white border from the edges of an image.
// Pixels within the whiteness threshold are considered background and removed.
func trimWhiteBorder(sourceImage image.Image) image.Image {
	imageBounds := sourceImage.Bounds()
	minX, minY := imageBounds.Max.X, imageBounds.Max.Y
	maxX, maxY := imageBounds.Min.X, imageBounds.Min.Y

	// Threshold scaled to 16-bit color values (uint32 range 0–65535)
	whitenessThreshold := uint32(10 * 257)

	for pixelY := imageBounds.Min.Y; pixelY < imageBounds.Max.Y; pixelY++ {
		for pixelX := imageBounds.Min.X; pixelX < imageBounds.Max.X; pixelX++ {
			redChannel, greenChannel, blueChannel, _ := sourceImage.At(pixelX, pixelY).RGBA()
			// A pixel is non-white if any channel falls below the white threshold
			if redChannel < 0xffff-whitenessThreshold ||
				greenChannel < 0xffff-whitenessThreshold ||
				blueChannel < 0xffff-whitenessThreshold {
				if pixelX < minX {
					minX = pixelX
				}
				if pixelY < minY {
					minY = pixelY
				}
				if pixelX > maxX {
					maxX = pixelX
				}
				if pixelY > maxY {
					maxY = pixelY
				}
			}
		}
	}

	if maxX < minX || maxY < minY {
		return sourceImage // nothing to trim
	}

	type subImager interface {
		SubImage(rectangle image.Rectangle) image.Image
	}
	if subImageSupported, ok := sourceImage.(subImager); ok {
		return subImageSupported.SubImage(image.Rect(minX, minY, maxX+1, maxY+1))
	}

	return sourceImage
}

// ScreenshotSheet takes a public Google Sheets URL and returns a PNG-encoded byte slice
// of the sheet's non-empty content area, preserving all Google-rendered formatting.
//
// Returns ErrSheetNotPublic if the sheet is private or requires authentication.
func ScreenshotSheet(requestContext context.Context, rawSheetURL string, sheetsAPIKey string) ([]byte, error) {
	spreadsheetID, sheetGID, parseError := parseSheetURL(rawSheetURL)
	if parseError != nil {
		return nil, parseError
	}

	usedCellRange, rangeError := getUsedCellRange(requestContext, spreadsheetID, sheetGID, sheetsAPIKey)
	if rangeError != nil {
		return nil, rangeError
	}

	pdfBytes, exportError := exportSheetAsPDF(spreadsheetID, sheetGID, usedCellRange)
	if exportError != nil {
		return nil, exportError
	}

	return renderPDFPageToTrimmedPNG(pdfBytes)
}
