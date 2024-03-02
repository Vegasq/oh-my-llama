package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"log"
	"net/http"
	"strings"
)

type APIResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Message   struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
}

type APIError struct {
	Error string `json:"error"`
}

type OMLApp struct {
	ModelName   string
	MainWindow  *fyne.Window
	ModelNames  []string
	ChatHistory *widget.Entry
	InputField  *widget.Entry
	APIHost     string
}

func NewOMLApp() *OMLApp {
	omlApp := app.New()
	window := omlApp.NewWindow("Oh My Llama")

	return &OMLApp{
		MainWindow:  &window,
		ModelNames:  []string{"gemma", "llama2", "mistral", "codellama"},
		ChatHistory: widget.NewMultiLineEntry(),
		InputField:  widget.NewMultiLineEntry(),
		APIHost:     "http://localhost:11434",
	}
}

func (oml *OMLApp) SetupUI() {
	oml.InputField.SetPlaceHolder("Type your query...")
	oml.ChatHistory.Wrapping = fyne.TextWrapWord

	modelNameInput := widget.NewSelect(oml.ModelNames, func(name string) {
		oml.ModelName = name
		oml.PullModel()
	})
	modelNameInput.SetSelected(oml.ModelNames[0])

	sendButton := widget.NewButton("Send", func() {
		oml.SendMessage(oml.InputField.Text)
		oml.InputField.SetText("") // Clear the input after sending
	})

	settingsButton := widget.NewButton("Settings", func() {
		oml.ShowSettings()
	})

	header := container.NewVBox(modelNameInput, settingsButton)
	footer := container.NewVBox(oml.InputField, sendButton)
	borderLayout := layout.NewBorderLayout(header, footer, nil, nil)
	content := container.New(borderLayout, header, oml.ChatHistory, footer)

	(*oml.MainWindow).SetContent(content)
	(*oml.MainWindow).Resize(fyne.NewSize(400, 600))
	(*oml.MainWindow).ShowAndRun()
}

func (oml *OMLApp) ShowSettings() {
	w := fyne.CurrentApp().NewWindow("Settings")

	hostEntry := widget.NewEntry()
	hostEntry.SetText(oml.APIHost)

	modelList := widget.NewList(
		func() int {
			return len(oml.ModelNames)
		},
		func() fyne.CanvasObject {
			hbox := container.NewHBox(widget.NewLabel(""), widget.NewButton("Remove", func() {}))
			return hbox
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*fyne.Container).Objects[0].(*widget.Label).SetText(oml.ModelNames[i])
			o.(*fyne.Container).Objects[1].(*widget.Button).OnTapped = func() {
				oml.ModelNames = append(oml.ModelNames[:i], oml.ModelNames[i+1:]...)
			}
		},
	)

	addModelEntry := widget.NewEntry()
	addModelEntry.SetPlaceHolder("Add new model")

	addButton := widget.NewButton("Add", func() {
		if addModelEntry.Text != "" {
			oml.ModelNames = append(oml.ModelNames, addModelEntry.Text)
			modelList.Refresh()
			addModelEntry.SetText("") // Clear the entry after adding
		}
	})

	saveButton := widget.NewButton("Save", func() {
		oml.APIHost = hostEntry.Text
		if strings.TrimSuffix(oml.APIHost, "/") != oml.APIHost {
			oml.APIHost = strings.TrimSuffix(oml.APIHost, "/")
		}
		w.Close()
	})

	cancelButton := widget.NewButton("Cancel", func() {
		w.Close()
	})

	content := container.NewVBox(
		widget.NewLabel("API Host"),
		hostEntry,
		addModelEntry,
		container.NewHBox(addButton),
		widget.NewLabel("Models"),
		modelList,
		container.NewHBox(saveButton, cancelButton),
	)

	w.SetContent(content)
	w.Resize(fyne.NewSize(300, 400)) // Adjusted for additional content
	w.Show()
}

func (oml *OMLApp) PullModel() {
	url := oml.APIHost + "/api/pull"
	payload := map[string]string{"name": oml.ModelName}
	bytesRepresentation, err := json.Marshal(payload)
	if err != nil {
		log.Fatalln(err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(bytesRepresentation))
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()
		oml.ChatHistory.Text += line + "\n"
		oml.ChatHistory.Refresh()

		// Scroller to the bottom
		focusedItem := (*oml.MainWindow).Canvas().Focused()
		if focusedItem == nil || focusedItem != oml.ChatHistory {
			oml.ChatHistory.CursorRow = len(oml.ChatHistory.Text) - 1 // Sets the cursor to the end
		}
	}
}

func (oml *OMLApp) SendMessage(message string) {
	url := oml.APIHost + "/api/chat"
	payload := map[string]interface{}{
		"model":    oml.ModelName,
		"messages": []map[string]string{{"role": "user", "content": message}},
	}
	bytesRepresentation, err := json.Marshal(payload)
	if err != nil {
		log.Fatalln(err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(bytesRepresentation))
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()

		var apiError APIError
		json.Unmarshal([]byte(line), &apiError)
		if apiError.Error != "" {
			fmt.Println(apiError.Error, "!")
			oml.ChatHistory.Text += apiError.Error + "\n"
			oml.ChatHistory.Refresh()

			return
		}

		var response APIResponse
		err = json.Unmarshal([]byte(line), &response)
		if err != nil {
			log.Println("Error unmarshaling response line:", err)
			continue
		}
		if response.Message.Role == "assistant" {
			oml.ChatHistory.Text += response.Message.Content
			oml.ChatHistory.Refresh()

			// Scroller to the bottom
			focusedItem := (*oml.MainWindow).Canvas().Focused()
			if focusedItem == nil || focusedItem != oml.ChatHistory {
				oml.ChatHistory.CursorRow = len(oml.ChatHistory.Text) - 1 // Sets the cursor to the end
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalln("Error reading response:", err)
	}
}

func main() {
	oml := NewOMLApp()
	oml.SetupUI()
}
