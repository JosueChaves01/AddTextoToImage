package main

import (
	"bytes"
	"errors"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
	"github.com/jordan-wright/email"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	"image"
	"image/jpeg"
	"image/png"
	"net/smtp"
	"os"
)

// AppState Struct to hold application state
type AppState struct {
	TextTop       string
	TextBottom    string
	SelectedColor string
	ImgPath       string
	OutputName    string
	FontPath      string
	EmailAddr     string
	ImgFormat     string
	ImageWidget   *canvas.Image
	OriginalImage image.Image
	ModifiedImage image.Image
}

// Custom widget for displaying text with fixed width
type fixedWidthLabel struct {
	widget.Label
}

func newFixedWidthLabel(text string) *fixedWidthLabel {
	label := &fixedWidthLabel{}
	label.ExtendBaseWidget(label)
	label.SetText(text)
	return label
}

func (l *fixedWidthLabel) MinSize() fyne.Size {
	return fyne.NewSize(300, l.Label.MinSize().Height)
}

// Function to calculate font size based on image dimensions and text
func calculateFontSize(image *gg.Context, fontPath string, text string) float64 {
	//obtain max width and height for text based on image dimensions
	maxTextWidth := float64(image.Image().Bounds().Size().X)
	maxTextHeight := float64(image.Image().Bounds().Size().Y)

	maxFontSize := 200
	fontSize := float64(maxFontSize)

	for {
		err := image.LoadFontFace(fontPath, fontSize)
		if err != nil {
			return 0
		}
		textWidth, textHeight := image.MeasureString(text)

		if textWidth <= maxTextWidth && textHeight <= maxTextHeight {
			break
		}

		fontSize -= 1
		if fontSize <= 0 {
			break
		}
	}

	return fontSize
}

// Function to add text to image
func (state *AppState) addTextToImage(window fyne.Window) {
	img := imaging.Clone(state.OriginalImage)

	dc := gg.NewContextForImage(img)
	imgWidth := float64(img.Bounds().Size().X)
	imgHeight := float64(img.Bounds().Size().Y)

	if state.SelectedColor == "Black" {
		dc.SetHexColor("#000000")
	} else if state.SelectedColor == "White" {
		dc.SetHexColor("#FFFFFF")
	}

	drawText := func(text string, posY float64) {
		fontSize := calculateFontSize(dc, state.FontPath, text)
		if err := dc.LoadFontFace(state.FontPath, fontSize); err != nil {
			panic(err)
		}
		textWidth, _ := dc.MeasureString(text)
		textX := (imgWidth - textWidth) / 2
		dc.DrawStringAnchored(text, textX, posY, 0, 0)
	}

	if state.TextTop != "" {
		drawText(state.TextTop, imgHeight/5)
	}

	if state.TextBottom != "" {
		drawText(state.TextBottom, imgHeight/1.1)
	}

	// Update the modified image in the state
	state.ModifiedImage = dc.Image()

	// Replace the image in the canvas with the modified image
	state.ImageWidget.Image = state.ModifiedImage

	// Refresh the canvas
	state.ImageWidget.Refresh()
	window.Canvas().Refresh(state.ImageWidget)
}

// Function to save image to file
func saveImage(img image.Image, outputPath string, imgFormat string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)

	switch imgFormat {
	case "png":
		err = png.Encode(file, img)
	case "jpeg":
		err = jpeg.Encode(file, img, nil)
	case "tiff":
		err = tiff.Encode(file, img, nil)
	case "bmp":
		err = bmp.Encode(file, img)
	default:
		err = errors.New("unsupported image format")
	}

	return err
}

// Function to send email with image attachment
func sendEmail(to string, img image.Image, outputName string, imgFormat string) error {
	// Create a new email message
	e := email.NewEmail()
	e.From = "your_email@gmail.com" // Replace with your email
	e.To = []string{to}
	e.Subject = "Image Attachment"

	var imgData bytes.Buffer
	switch imgFormat {
	case "png":
		// Encode image to PNG format
		err := png.Encode(&imgData, img)
		if err != nil {
			return err
		}
	case "jpeg":
		// Encode image to JPG format
		err := jpeg.Encode(&imgData, img, nil)
		if err != nil {
			return err
		}
	case "tiff":
		// Encode image to TIFF format
		err := tiff.Encode(&imgData, img, nil)
		if err != nil {
			return err
		}
	case "bmp":
		// Encode image to BMP format
		err := bmp.Encode(&imgData, img)
		if err != nil {
			return err
		}
	default:
		return errors.New("unsupported image format")
	}

	// Attach the image
	reader := bytes.NewReader(imgData.Bytes())
	_, err := e.Attach(reader, outputName+"."+imgFormat, outputName+"/"+imgFormat)
	if err != nil {
		return err
	}

	// Define SMTP server configuration
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"
	smtpUser := "josuchazu.jecz@gmail.com"
	smtpPassword := "efanihcnknulezeu"

	// Connect to the SMTP server
	auth := smtp.PlainAuth("", smtpUser, smtpPassword, smtpHost)
	if err := e.Send(smtpHost+":"+smtpPort, auth); err != nil {
		return err
	}

	fmt.Println("Email sent successfully")
	return nil
}

// Function to create the main UI
func (state *AppState) makeUI(app fyne.App) fyne.Window {
	window := app.NewWindow("Image Text Adder")
	window.Resize(fyne.NewSize(800, 600))
	window.CenterOnScreen()

	// Establece el ícono de la aplicación
	iconPath := "Logo.png" // Reemplaza esto con la ruta real de tu archivo de ícono

	// Abre el archivo de imagen
	file, err := os.Open(iconPath)
	if err != nil {
		return nil
	}
	defer file.Close()

	logo, _, _ := image.Decode(file)
	// Convierte la imagen en bytes
	buffer := new(bytes.Buffer)
	err = imaging.Encode(buffer, logo, 1)
	if err != nil {
		return nil
	}
	imgResource := fyne.NewStaticResource("Logo.png", buffer.Bytes())
	window.SetIcon(imgResource)

	//create new Im
	imgWidget := canvas.NewImageFromFile(state.ImgPath)
	imgWidget.FillMode = canvas.ImageFillContain

	fontSelector := widget.NewSelect([]string{"creamy", "American", "Typewriter"}, func(s string) {
		state.FontPath = s + ".ttf"
	})

	colorSelector := widget.NewSelect([]string{"Black", "White"}, func(s string) {
		state.SelectedColor = s
	})

	formatSelector := widget.NewSelect([]string{"png", "jpeg", "tiff", "bmp"}, func(s string) {
		state.ImgFormat = s
	})

	textTopEntry := widget.NewEntry()
	textBottomEntry := widget.NewEntry()
	outputEntry := widget.NewEntry()
	emailEntry := widget.NewEntry()

	// Custom widget for displaying selected image path
	imgPathLabel := newFixedWidthLabel("")
	imgPathLabel.SetText("Select an image!!")

	// Button to trigger file dialog for image selection
	selectImgButton := widget.NewButton("Select Image", func() {
		fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err == nil && reader != nil {
				// Get the file path from URI
				state.ImgPath = reader.URI().Path()
				imgPathLabel.SetText(state.ImgPath)

				// Open the original image
				img, err := imaging.Open(state.ImgPath)
				if err != nil {
					dialog.ShowError(err, window)
					return
				}

				// Get the dimensions of the original image
				imgWidth := float64(img.Bounds().Size().X)
				imgHeight := float64(img.Bounds().Size().Y)

				// Define the minimum and maximum dimensions
				minWidth := 1200
				minHeight := 1200
				maxWidth := 1920
				maxHeight := 1080

				// Resize the image if it falls outside the desired size range
				if imgWidth < float64(minWidth) || imgHeight < float64(minHeight) || imgWidth > float64(maxWidth) || imgHeight > float64(maxHeight) {
					// Calculate the aspect ratio
					aspectRatio := imgWidth / imgHeight

					// Determine the target width and height based on aspect ratio and desired dimensions
					targetWidth := imgWidth
					targetHeight := imgHeight

					if imgWidth < float64(minWidth) || imgHeight < float64(minHeight) {
						// If image is smaller than the minimum size, resize to minimum size
						if aspectRatio > float64(maxWidth)/float64(maxHeight) {
							targetWidth = float64(minWidth)
							targetHeight = float64(minHeight)
						} else {
							targetHeight = float64(minHeight)
							targetWidth = float64(maxHeight)
						}
					} else {
						// If image is larger than the maximum size, resize to maximum size
						if aspectRatio > float64(maxWidth)/float64(maxHeight) {
							targetHeight = float64(maxHeight)
							targetWidth = targetHeight * aspectRatio
						} else {
							targetWidth = float64(maxWidth)
							targetHeight = targetWidth / aspectRatio
						}
					}

					// Resize the image
					img = imaging.Resize(img, int(targetWidth), int(targetHeight), imaging.Lanczos)
				}

				// Set the original image in the state
				state.OriginalImage = img

				// Set the image widget with the resized image
				imgWidget.Image = state.OriginalImage
				state.ImageWidget = imgWidget
				imgWidget.Refresh()
			}
		}, window)
		fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".png", ".jpg", ".jpeg", ".tif", ".bmp"}))
		fileDialog.Show()
	})

	// Button to exit the application
	exitButton := widget.NewButton("Exit", func() {
		app.Quit()
	})

	addTextButton := widget.NewButton("Add Text to Image", func() {
		// Update output name from entry
		state.OutputName = outputEntry.Text

		// Check if image path is provided
		if state.ImgPath == "" {
			dialog.ShowError(errors.New("please select an image"), window)
			return
		}
		// Check if font path is provided
		if state.FontPath == "" {
			dialog.ShowError(errors.New("please select a font"), window)
			return
		}
		// Check if color is selected
		if state.SelectedColor == "" {
			dialog.ShowError(errors.New("please select a color"), window)
			return
		}

		// Add text to image
		state.TextTop = textTopEntry.Text
		state.TextBottom = textBottomEntry.Text
		state.addTextToImage(window)
	})

	saveButton := widget.NewButton("Save", func() {
		// Update output name from entry
		state.OutputName = outputEntry.Text

		// Check if output name is provided
		if state.OutputName == "" {
			dialog.ShowError(errors.New("please specify an output name"), window)
			return
		}

		// Check if image path is provided
		if state.ImgPath == "" {
			dialog.ShowError(errors.New("please select an image"), window)
			return
		}

		// Show file save dialog
		saveFileDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err == nil && writer != nil {
				defer func(writer fyne.URIWriteCloser) {
					err := writer.Close()
					if err != nil {

					}
				}(writer)

				// Save the image to the selected file location
				err := saveImage(state.ModifiedImage, writer.URI().Path(), state.ImgFormat)
				if err != nil {
					dialog.ShowError(err, window)
					return
				}

				dialog.ShowInformation("Image Saved", "Image has been saved successfully.", window)
			}
		}, window)
		saveFileDialog.SetFileName(state.OutputName + "." + state.ImgFormat)
		saveFileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".png", ".jpg", ".jpeg", ".tiff"}))
		saveFileDialog.Show()
	})

	sendEmailButton := widget.NewButton("Send Email", func() {
		// Update email address from entry
		state.EmailAddr = emailEntry.Text
		state.OutputName = outputEntry.Text

		// Check if email address is provided
		if state.EmailAddr == "" {
			dialog.ShowError(errors.New("please enter an email address"), window)
			return
		}

		// Check if the output name is provided
		if state.OutputName == "" {
			dialog.ShowError(errors.New("please specify an output name"), window)
			return
		}

		// Check if image path is provided
		if state.ImgPath == "" {
			dialog.ShowError(errors.New("please select an image"), window)
			return
		}

		// Send email with the image attached
		if err := sendEmail(state.EmailAddr, state.ModifiedImage, state.OutputName, state.ImgFormat); err != nil {
			fmt.Println(err)
			dialog.ShowError(err, window)
			return
		}

		dialog.ShowInformation("Email Sent", "Image has been sent to "+state.EmailAddr, window)
	})

	// Agregar el logo al contenido de la ventana
	controlsContainer := container.NewVBox(
		widget.NewLabel("Select Format:"),
		formatSelector,
		widget.NewLabel("Select Font:"),
		fontSelector,
		widget.NewLabel("Select Color:"),
		colorSelector,
		widget.NewLabel("Top Text:"),
		textTopEntry,
		widget.NewLabel("Bottom Text:"),
		textBottomEntry,
		widget.NewLabel("Select Image:"),
		container.NewHBox(imgPathLabel, selectImgButton),
		widget.NewLabel("Output Name:"),
		outputEntry,
		widget.NewLabel("Email Address:"),
		emailEntry,
		container.NewHBox(addTextButton, saveButton, sendEmailButton, exitButton),
	)

	split := container.NewHSplit(
		controlsContainer,
		imgWidget,
	)
	split.Offset = 0.3

	window.SetContent(split)
	window.ShowAndRun()

	return window
}

func main() {
	// Create a new instance of the AppState struct
	state := &AppState{}

	// Initialize and run the Fyne application
	a := app.New()
	state.makeUI(a)
}
