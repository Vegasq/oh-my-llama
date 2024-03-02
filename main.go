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
}

func NewOMLApp() *OMLApp {
	omlApp := app.New()
	window := omlApp.NewWindow("Oh My Llama")

	return &OMLApp{
		MainWindow:  &window,
		ModelNames:  []string{"gemma", "llama2", "mistral", "codellama"},
		ChatHistory: widget.NewMultiLineEntry(),
		InputField:  widget.NewMultiLineEntry(),
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

	header := container.NewVBox(modelNameInput)
	footer := container.NewVBox(oml.InputField, sendButton)
	borderLayout := layout.NewBorderLayout(header, footer, nil, nil)
	content := container.New(borderLayout, header, oml.ChatHistory, footer)

	(*oml.MainWindow).SetContent(content)
	(*oml.MainWindow).Resize(fyne.NewSize(400, 600))
	(*oml.MainWindow).ShowAndRun()
}

func (oml *OMLApp) PullModel() {
	url := "http://localhost:11434/api/pull"
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
	url := "http://localhost:11434/api/chat"
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
