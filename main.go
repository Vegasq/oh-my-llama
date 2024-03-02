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

type OML struct {
	modelNameWidget string
	window          *fyne.Window
}

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

func main() {
	oml := OML{}
	omlApp := app.New()
	window := omlApp.NewWindow("Oh My LLama")

	oml.window = &window

	input := widget.NewMultiLineEntry()
	input.SetPlaceHolder("Type your query...")
	chatHistory := widget.NewMultiLineEntry()
	chatHistory.Wrapping = fyne.TextWrapWord

	modelNames := []string{"gemma", "llama2", "mistral", "codellama"}

	modelNameInput := widget.NewSelect(modelNames, func(name string) {
		oml.modelNameWidget = name
		go pullModel(chatHistory, &oml)
	})
	modelNameInput.SetSelected(modelNames[0])

	sendButton := widget.NewButton("Send", func() {
		go sendMessage(input.Text, chatHistory, &oml)
		input.SetText("") // Clear the input after sending
	})
	header := container.NewVBox(modelNameInput)
	footer := container.NewVBox(input, sendButton)
	borderLayout := layout.NewBorderLayout(header, footer, nil, nil)
	content := container.New(borderLayout, header, chatHistory, footer)

	window.SetContent(content)
	window.Resize(fyne.NewSize(400, 600))
	window.ShowAndRun()
}

func pullModel(entry *widget.Entry, oml *OML) {
	url := "http://localhost:11434/api/pull"
	payload := map[string]string{"name": oml.modelNameWidget}
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
		entry.Text += line + "\n"
		entry.Refresh()

		// Scroller to the bottom
		focusedItem := (*oml.window).Canvas().Focused()
		if focusedItem == nil || focusedItem != entry {
			entry.CursorRow = len(entry.Text) - 1 // Sets the cursor to the end
		}
	}
}

func sendMessage(message string, entry *widget.Entry, oml *OML) {
	url := "http://localhost:11434/api/chat"
	payload := map[string]interface{}{
		"model":    oml.modelNameWidget,
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
			entry.Text += apiError.Error + "\n"
			entry.Refresh()

			return
		}

		var response APIResponse
		err = json.Unmarshal([]byte(line), &response)
		if err != nil {
			log.Println("Error unmarshaling response line:", err)
			continue
		}
		if response.Message.Role == "assistant" {
			entry.Text += response.Message.Content
			entry.Refresh()

			// Scroller to the bottom
			focusedItem := (*oml.window).Canvas().Focused()
			if focusedItem == nil || focusedItem != entry {
				entry.CursorRow = len(entry.Text) - 1 // Sets the cursor to the end
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalln("Error reading response:", err)
	}
}
