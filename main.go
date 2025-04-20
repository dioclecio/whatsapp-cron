package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings" // Add this import
	"time"
	"math/rand"
	"github.com/playwright-community/playwright-go"
	"unicode/utf8"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	"image/png"
)

const (
	dbFile = "data/mensagens.json"
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
	log.Println("Inicializando o Playwright...")
	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("Não foi possível iniciar o Playwright: %v", err)
	}
	if pw == nil {
		log.Fatalf("Playwright não foi inicializado corretamente")
	}
	defer pw.Stop()

	// Verifica e instala apenas o driver do Chromium
	if err := playwright.Install(&playwright.RunOptions{
		Browsers: []string{"chromium-headless-shell"},
	}); err != nil {
		log.Fatalf("Erro ao instalar o driver do Playwright (Chromium): %v", err)
	}

	log.Println("Iniciando o navegador...")
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
		Args: []string{
				"--no-sandbox",
				"--disable-setuid-sandbox",
				"--disable-dev-shm-usage",
				"--disable-accelerated-2d-canvas",
				"--no-first-run",
				"--no-zygote",
				"--disable-gpu",
				"--deterministic-fetch",
				"--disable-features=IsolateOrigins,site-per-process",
		},
	})
	if err != nil {
		log.Fatalf("Não foi possível iniciar o navegador: %v", err)
	}
	defer browser.Close()

	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		NoViewport: playwright.Bool(true),
		UserAgent: playwright.String("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36"),
		
	})
	if err != nil {
		log.Fatalf("Não foi possível criar o contexto do navegador: %v", err)
	}
	defer context.Close()

	log.Println("Criando uma nova página...")
	page, err := browser.NewPage()
	if err != nil {
		log.Fatalf("Não foi possível criar uma nova página: %v", err)
	} else {
		log.Println("Página criada com sucesso.")
	}

	log.Println("Navegando para o WhatsApp Web...")
	if _, err := page.Goto("https://web.whatsapp.com", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(2147483647), // Aumenta o timeout para 24 dias
	}); err != nil {
		log.Fatalf("Erro ao abrir WhatsApp: %v", err)
	} else {
		log.Println("WhatsApp Web carregado com sucesso.")
	}

	log.Printf("Aguarde enquanto o WhatsApp Web carrega...")
	time.Sleep(20 * time.Second)
	fmt.Println("Escaneie o QR Code. Você tem 1 minuto.")

	// Garante que o diretório 'data' existe
	if _, err := os.Stat("data"); os.IsNotExist(err) {
		if err := os.Mkdir("data", 0755); err != nil {
			log.Fatalf("Erro ao criar o diretório 'data': %v", err)
		}
	}

	// Tira um screenshot e salva no diretório especificado
	if _, err := page.Screenshot(playwright.PageScreenshotOptions{
		Path: playwright.String("data/qrcode.png"), // Salva no diretório data
		FullPage: playwright.Bool(true),           // Captura a página inteira
	}); err != nil {
		log.Printf("Erro ao tirar screenshot: %v", err)
	} else {
		log.Println("Screenshot capturado com sucesso: data/qrcode.png")
		// Converte o QR code para ASCII
		if err := displayQRCodeASCII("data/qrcode.png"); err != nil {
			log.Printf("Erro ao converter QR code para ASCII: %v", err)
		}
	}

	time.Sleep(1 * time.Minute)

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

	// Divide a mensagem em partes usando "\r" como separador
	parts := strings.Split(encodedMsg, "\r")

	// Simula a digitação da mensagem, inserindo Shift+Enter entre as partes
	for i, part := range parts {
		if err := msgBox.Type(part); err != nil {
			return fmt.Errorf("erro ao digitar a mensagem: %v", err)
		}
		if i < len(parts)-1 {
			// Simula Shift+Enter para criar uma nova linha
			if err := msgBox.Press("Shift+Enter"); err != nil {
				return fmt.Errorf("erro ao pressionar Shift+Enter: %v", err)
			}
		}
	}

	// Pressiona Enter para enviar a mensagem
	if err := msgBox.Press("Enter"); err != nil {
		return fmt.Errorf("erro ao pressionar Enter: %v", err)
	}

	log.Printf("Mensagem enviada para '%s': %s", destino, msg)
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
	// Abre o arquivo de imagem
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("erro ao abrir o arquivo de imagem: %v", err)
	}
	defer file.Close()

	// Decodifica a imagem PNG
	img, err := png.Decode(file)
	if err != nil {
		return fmt.Errorf("erro ao decodificar a imagem PNG: %v", err)
	}

	// Converte a imagem para um bitmap binário
	bitmap, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return fmt.Errorf("erro ao criar o bitmap binário: %v", err)
	}

	// Cria um leitor de QR code
	reader := qrcode.NewQRCodeReader()
	result, err := reader.Decode(bitmap, nil)
	if err != nil {
		return fmt.Errorf("erro ao decodificar o QR code: %v", err)
	}

	// Obtém o conteúdo do QR code
	qrContent := result.String()

	// Limpa o terminal para melhor visibilidade
	fmt.Print("\033[H\033[2J")

	// Print header with box drawing characters
	fmt.Println("\n┌" + strings.Repeat("─", 102) + "┐")
	fmt.Println("│" + strings.Repeat(" ", 34) + "QR Code - Whatsapp Web" + strings.Repeat(" ", 34) + "│")
	fmt.Println("│" + strings.Repeat(" ", 28) + "Escaneie usando seu smartphone" + strings.Repeat(" ", 28) + "│")
	fmt.Println("├" + strings.Repeat("─", 102) + "┤")

	// Create simple ASCII QR representation
	size := 15 // Reduzido para ajustar as dimensões do QR Code
	matrix, err := qrcode.NewQRCodeWriter().Encode(
		qrContent,
		gozxing.BarcodeFormat_QR_CODE,
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
