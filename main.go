package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"
	"math/rand"
	"github.com/playwright-community/playwright-go"
	"unicode/utf8"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	"image/png"
	"strings"
)

const (
	dbFile = "data/mensagens.json"
	// PlaywrightEndpoint is the environment variable that contains the remote Playwright URL.
	PlaywrightEndpoint = "ws://playwright:3000" // Default value. Override with env var.
)

// Mensagem represents a message to be sent.
type Mensagem struct {
	ID          int    `json:"id"`
	Destinatario string `json:"destinatario"`
	Conteudos   []string `json:"conteudos"`
	UltimoEnvio string `json:"ultimo_envio"`
	HorarioEnvio string `json:"horario_envio"`
	DiaSemana   []time.Weekday `json:"dia_semana"`
}

// Database represents the JSON database.
type Database struct {
	Mensagens []Mensagem `json:"mensagens"`
}

// fileInfo stores the last modification time of the database file.
type fileInfo struct {
	lastMod time.Time
}

func main() {
	// Initialize Playwright
	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("Não foi possível iniciar o Playwright: %v", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
		Args: []string{"--start-maximized"},
	})
	if err != nil {
		log.Fatalf("Não foi possível iniciar o navegador: %v", err)
	}
	defer browser.Close()

	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		NoViewport: playwright.Bool(true),
		UserAgent: playwright.String("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	})
	if err != nil {
		log.Fatalf("Não foi possível criar o contexto do navegador: %v", err)
	}
	defer context.Close()

	page, err := context.NewPage()
	if err != nil {
		log.Fatalf("Não foi possível criar uma nova página: %v", err)
	}

	if _, err := page.Goto("https://web.whatsapp.com", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		log.Fatalf("Erro ao abrir WhatsApp: %v", err)
	}
	log.Printf("Aguarde enquanto o WhatsApp Web carrega...")
	time.Sleep(30 * time.Second)
	fmt.Println("Escaneie o QR Code. Você tem 2 minutos.")

	// Take a screenshot and convert to ASCII
	if _, err := page.Screenshot(playwright.PageScreenshotOptions{
		Path: playwright.String("data/qrcode.png"), // Save in the data directory
	}); err != nil {
		log.Printf("Erro ao tirar screenshot: %v", err)
	} else {
		// Convert QR code to ASCII
		if err := displayQRCodeASCII("data/qrcode.png"); err != nil {
			log.Printf("Erro ao converter QR code para ASCII: %v", err)
		}
	}

	time.Sleep(2 * time.Minute)

	log.Println("Tempo de escaneio vencido.")
	fileInfo := &fileInfo{}
	fileInfo.updateLastMod()

	for {
		if fileInfo.hasChanged() {
			log.Println("Arquivo mensagens.json foi modificado. Recarregando...")
			fileInfo.updateLastMod()
		}
		enviarMensagensNoHorario(page)
		time.Sleep(60 * time.Second)
	}
}

// loadDB loads the database from the JSON file.
func loadDB() (Database, error) {
	var db Database
	data, err := ioutil.ReadFile(dbFile)
	if err != nil {
		if os.IsNotExist(err) {
			db = Database{Mensagens: []Mensagem{}}
			if err := saveDB(db); err != nil {
				return db, err
			}
			return db, nil
		}
		return db, fmt.Errorf("erro ao ler o arquivo do banco de dados: %v", err)
	}

	err = json.Unmarshal(data, &db)
	if err != nil {
		return db, fmt.Errorf("erro ao decodificar o JSON do banco de dados: %v", err)
	}
	return db, nil
}

// saveDB saves the database to the JSON file.
func saveDB(db Database) error {
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return fmt.Errorf("erro ao codificar o JSON do banco de dados: %v", err)
	}

	err = ioutil.WriteFile(dbFile, data, 0644)
	if err != nil {
		return fmt.Errorf("erro ao salvar o arquivo do banco de dados: %v", err)
	}
	return nil
}

// enviarMensagensNoHorario checks if it's time to send messages and sends them.
func enviarMensagensNoHorario(page playwright.Page) {
	db, err := loadDB()
	if err != nil {
		log.Println("Erro ao carregar o banco de dados:", err)
		return
	}

	now := time.Now()
	currentTime := now.Format("15:04")
	currentDay := now.Weekday()

	for i, msg := range db.Mensagens {
		if msg.HorarioEnvio == currentTime {
			sendToday := false
			if len(msg.DiaSemana) == 0 {
				sendToday = true
			} else {
				for _, day := range msg.DiaSemana {
					if day == currentDay {
						sendToday = true
						break
					}
				}
			}

			if sendToday {
				if len(msg.Conteudos) == 0 {
					log.Println("Nenhuma mensagem definida para o destinatário:", msg.Destinatario)
					continue
				}
				if err := enviarViaPlaywright(page, msg.Destinatario, msg.Conteudos[rand.Intn(len(msg.Conteudos))]); err != nil {
					log.Println("Falha no envio:", err)
					continue
				}

				db.Mensagens[i].UltimoEnvio = now.Format(time.RFC3339)
				if err := saveDB(db); err != nil {
					log.Println("Erro ao salvar o banco de dados:", err)
				}
			}
		}
	}
}

func enviarViaPlaywright(page playwright.Page, destino, msg string) error {
	// Pressiona Esc para fechar menus ou pop-ups
	if err := page.Keyboard().Press("Escape"); err != nil {
		log.Println("Aviso: Não foi possível pressionar Esc:", err)
	}

	// Localiza e limpa o campo de pesquisa
	searchBox, err := page.WaitForSelector(`[contenteditable="true"][data-tab="3"]`, playwright.PageWaitForSelectorOptions{
		State: playwright.WaitForSelectorStateVisible,
	})
	if err != nil {
		return fmt.Errorf("erro ao encontrar a caixa de pesquisa: %v", err)
	}

	if err := searchBox.Click(); err != nil {
		return fmt.Errorf("erro ao clicar na caixa de pesquisa: %v", err)
	}

	if err := searchBox.Fill(""); err != nil {
		return fmt.Errorf("erro ao limpar a caixa de pesquisa: %v", err)
	}

	if err := searchBox.Type(destino); err != nil {
		return fmt.Errorf("erro ao digitar no campo de pesquisa: %v", err)
	}

	// Espera o chat aparecer e clica nele
	chatSelector := fmt.Sprintf(`span[title="%s"]`, destino)
	chat, err := page.WaitForSelector(chatSelector, playwright.PageWaitForSelectorOptions{
		State: playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(5000),
	})
	if err != nil {
		log.Printf("Aviso: Destino '%s' não encontrado na lista de chats. Verifique se o nome está correto e se o chat já foi iniciado.", destino)
		return nil
	}

	if err := chat.Click(); err != nil {
		return fmt.Errorf("erro ao clicar no chat: %v", err)
	}

	// Localiza e preenche o campo de mensagem
	msgBox, err := page.WaitForSelector(`[contenteditable="true"][data-tab="10"]`, playwright.PageWaitForSelectorOptions{
		State: playwright.WaitForSelectorStateVisible,
	})
	if err != nil {
		return fmt.Errorf("erro ao encontrar a caixa de mensagem: %v", err)
	}

	// Garante que a mensagem está em formato Unicode
	if !utf8.ValidString(msg) {
		msg = string([]rune(msg))
	}
	encoder := unicode.UTF8.NewEncoder()
	encodedMsg, _, err := transform.String(encoder, msg)
	if err != nil {
		return fmt.Errorf("erro ao codificar a mensagem para Unicode: %v", err)
	}

	if err := msgBox.Fill(encodedMsg); err != nil {
		return fmt.Errorf("erro ao preencher a caixa de mensagem: %v", err)
	}

	if err := msgBox.Press("Enter"); err != nil {
		return fmt.Errorf("erro ao pressionar Enter: %v", err)
	}

	return nil
}

// updateLastMod updates the last modification time of the database file.
func (fi *fileInfo) updateLastMod() {
	info, err := os.Stat(dbFile)
	if err != nil {
		log.Println("Erro ao obter informações do arquivo:", err)
		return
	}
	fi.lastMod = info.ModTime()
}

// hasChanged checks if the database file has been modified since the last check.
func (fi *fileInfo) hasChanged() bool {
	info, err := os.Stat(dbFile)
	if err != nil {
		log.Println("Erro ao obter informações do arquivo:", err)
		return false
	}
	return info.ModTime().After(fi.lastMod)
}

// displayQRCodeASCII converts a QR code image to ASCII art and displays it.
func displayQRCodeASCII(filepath string) error {
	// Open and decode the PNG file
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("erro ao abrir arquivo de imagem: %v", err)
	}
	defer file.Close()

	// Decode PNG image
	img, err := png.Decode(file)
	if err != nil {
		return fmt.Errorf("erro ao decodificar imagem PNG: %v", err)
	}

	// Convert image to binary bitmap
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return fmt.Errorf("erro ao criar bitmap: %v", err)
	}

	// Create QR reader
	reader := qrcode.NewQRCodeReader()
	result, err := reader.Decode(bmp, nil)
	if err != nil {
		return fmt.Errorf("erro ao decodificar QR code: %v", err)
	}

	// Get QR code content
	qrContent := result.String()

	// Clear terminal for better visibility
	fmt.Print("\033[H\033[2J")

	// Print header with box drawing characters
	fmt.Println("\n┌" + strings.Repeat("─", 102) + "┐")
	fmt.Println("│" + strings.Repeat(" ", 34) + "QR Code - Whatsapp Web" + strings.Repeat(" ", 34) + "│")
	fmt.Println("│" + strings.Repeat(" ", 28) + "Escaneie usando seu smartphone" + strings.Repeat(" ", 28) + "│")
	fmt.Println("├" + strings.Repeat("─", 102) + "┤")

	// Create simple ASCII QR representation
	size := 25 // Adjust size as needed
	matrix, err := qrcode.NewQRCodeWriter().Encode(
		qrContent, 
		gozxing.BarcodeFormat_QR_CODE, // Changed from qrcode.BarcodeFormat_QR_CODE
		size, 
		size, 
		nil,
	)
	if err != nil {
		return fmt.Errorf("erro ao gerar QR code ASCII: %v", err)
	}

	// Print QR code with borders
	for y := 0; y < matrix.GetHeight(); y++ {
		fmt.Print("│ " + strings.Repeat(" ", 2))
		for x := 0; x < matrix.GetWidth(); x++ {
			if matrix.Get(x, y) {
				fmt.Print("██")
			} else {
				fmt.Print("  ")
			}
		}
		fmt.Println(strings.Repeat(" ", 2) + "│")
	}

	// Print footer
	fmt.Println("└" + strings.Repeat("─", 102) + "┘")
	fmt.Println("Aguardando scan do QR Code...")

	return nil
}
