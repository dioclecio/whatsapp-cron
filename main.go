package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"
	"math/rand"
	"github.com/tebeka/selenium"
	"unicode/utf8"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	
)

const (
	seleniumPath = "chromedriver"  // Changed from geckodriver to chromedriver
	defaultPort  = 4444
	dbFile       = "data/mensagens.json" // JSON database file
)

// Mensagem represents a message to be sent.
type Mensagem struct {
	ID          int    `json:"id"`
	Destinatario string `json:"destinatario"`
	Conteudos   []string `json:"conteudos"`
	UltimoEnvio string `json:"ultimo_envio"`
	HorarioEnvio string `json:"horario_envio"`
	DiaSemana   []time.Weekday `json:"dia_semana"` // Add day of the week
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
	seleniumHub := os.Getenv("SELENIUM_HUB")
	var service *selenium.Service
	var err error
	var caps selenium.Capabilities
	time.Sleep(5 * time.Second)

	if seleniumHub == "" {
		log.Println("SELENIUM_HUB environment variable not set. Using local ChromeDriver.")
		service, err = selenium.NewChromeDriverService(seleniumPath, defaultPort)
		if err != nil {
			log.Fatal("Erro no ChromeDriver:", err)
		}
		defer service.Stop()
		seleniumHub = fmt.Sprintf("http://localhost:%d/wd/hub", defaultPort)
		caps = selenium.Capabilities{
			"browserName": "chrome",
			"goog:chromeOptions": map[string]interface{}{
				"args": []string{
					"--no-sandbox",
					"--disable-dev-shm-usage",
					"--disable-gpu",
					"--window-size=1920,1080",
					"--use-fake-ui-for-media-stream",
                    "--user-agent=Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
				},
				"prefs": map[string]interface{}{
					"profile.default_content_setting_values.notifications": 2,
				},
			},
		}
	} else {
		log.Printf("SELENIUM_HUB environment variable set. Using remote Selenium Hub: %s\n", seleniumHub)
		seleniumHub = fmt.Sprintf("http://%s/wd/hub", seleniumHub)
		caps = selenium.Capabilities{
			"browserName": "chrome",
			"goog:chromeOptions": map[string]interface{}{
				"args": []string{
					"--no-sandbox",
					"--disable-dev-shm-usage",
					"--disable-gpu",
					"--window-size=1920,1080",
					"--use-fake-ui-for-media-stream",
                    "--user-agent=Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
				},
				"prefs": map[string]interface{}{
					"profile.default_content_setting_values.notifications": 2,
				},
			},
		}
	}

	var wd selenium.WebDriver
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		wd, err = selenium.NewRemote(caps, seleniumHub)
		if err == nil {
			break // Success, exit the retry loop
		}
		log.Printf("Erro ao iniciar navegador (tentativa %d/%d): %v", i+1, maxRetries, err)
		if i < maxRetries-1 {
			log.Println("Tentando novamente em 60 segundos...")
			time.Sleep(60 * time.Second)
		}
	}
	if err != nil {
		log.Fatalf("Falha ao iniciar o navegador após %d tentativas. Encerrando.", maxRetries)
		return
	}
	defer func() {
		if wd != nil {
			wd.Quit()
		}
	}()

	// Configure timeout
	if err := wd.SetImplicitWaitTimeout(30 * time.Second); err != nil {
		log.Fatal("Erro ao configurar timeout:", err)
	}

	// Set page load timeout
	if err := wd.SetPageLoadTimeout(30 * time.Second); err != nil {
		log.Fatal("Erro ao configurar page load timeout:", err)
	}

	// Maximize window
	if err := wd.MaximizeWindow(""); err != nil {
		log.Println("Aviso: Não foi possível maximizar a janela:", err)
	}

	// Abre WhatsApp Web
	if err := wd.Get("https://web.whatsapp.com"); err != nil {
		log.Fatal("Erro ao abrir WhatsApp:", err)
	}

	fmt.Println("Escaneie o QR Code. Você tem 2 minutos.")
	time.Sleep(2 * time.Minute)

	log.Println("Tempo de escaneio vencido.")
	// Initialize file info
	fileInfo := &fileInfo{}
	fileInfo.updateLastMod()

	// Configura agendamento
	for {
		if fileInfo.hasChanged() {
			log.Println("Arquivo mensagens.json foi modificado. Recarregando...")
			fileInfo.updateLastMod()
		}
		enviarMensagensNoHorario(wd)
		time.Sleep(60 * time.Second) // Check every 60 seconds
	}
}

// loadDB loads the database from the JSON file.
func loadDB() (Database, error) {
	var db Database
	data, err := ioutil.ReadFile(dbFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Create an empty database if the file doesn't exist
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
func enviarMensagensNoHorario(wd selenium.WebDriver) {
	db, err := loadDB()
	if err != nil {
		log.Println("Erro ao carregar o banco de dados:", err)
		return
	}

	now := time.Now()
	currentTime := now.Format("15:04") // Format as HH:MM
	currentDay := now.Weekday()

	for i, msg := range db.Mensagens {
		if msg.HorarioEnvio == currentTime {
			// Check if the current day is in the list of allowed days
			sendToday := false
			if len(msg.DiaSemana) == 0 {
				sendToday = true // If no days are specified, send every day
			} else {
				for _, day := range msg.DiaSemana {
					if day == currentDay {
						sendToday = true
						break
					}
				}
			}

			if sendToday {
				// Select a random message from the array
				if len(msg.Conteudos) == 0 {
							log.Println("Nenhuma mensagem definida para o destinatário:", msg.Destinatario)
							continue
				}
				if err := enviarViaSelenium(wd, msg.Destinatario, msg.Conteudos[rand.Intn(len(msg.Conteudos))]); err != nil {
					log.Println("Falha no envio:", err)
					continue
				}

				db.Mensagens[i].UltimoEnvio = now.Format(time.RFC3339) // Update last sent time
				if err := saveDB(db); err != nil {
					log.Println("Erro ao salvar o banco de dados:", err)
				}
			}
		}
	}
}

func isSameDay(dateStr string, now time.Time) bool {
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return false
	}
	return now.Year() == t.Year() && now.YearDay() == t.YearDay()
}

func enviarViaSelenium(wd selenium.WebDriver, destino, msg string) error {
	// Press Esc to close any open menus or pop-ups
	body, err := wd.FindElement(selenium.ByTagName, "body")
	if err == nil {
		body.SendKeys(selenium.EscapeKey)
	}

	// Localiza campo de pesquisa (Chromium XPATH)
	searchBox, err := wd.FindElement(selenium.ByXPATH, `//div[@contenteditable="true"][@data-tab="3"]`)
	if err != nil {
		return fmt.Errorf("erro ao encontrar a caixa de pesquisa: %v", err)
	}
	searchBox.Clear()
	searchBox.SendKeys(destino)

	// Wait for the chat to appear in the list
	time.Sleep(5 * time.Second)

	// Find chat using Chromium XPATH
	chat, err := wd.FindElement(selenium.ByXPATH, fmt.Sprintf(`//span[@title="%s"]`, destino))
	if err != nil {
		log.Printf("Aviso: Destino '%s' não encontrado na lista de chats. Verifique se o nome está correto e se o chat já foi iniciado.", destino)
		return nil
	}
	chat.Click()

	// Wait for the message box to be ready
	time.Sleep(5 * time.Second)

	// Localiza campo de mensagem (Chromium XPATH)
	msgBox, err := wd.FindElement(selenium.ByXPATH, `//div[@contenteditable="true"][@data-tab="10"]`)
	if err != nil {
		return fmt.Errorf("erro ao encontrar a caixa de mensagem: %v", err)
	}

	// Ensure the message is in Unicode format and encode it
	if !utf8.ValidString(msg) {
		msg = string([]rune(msg)) // Convert to a valid UTF-8 string
	}
	encoder := unicode.UTF8.NewEncoder()
	encodedMsg, _, err := transform.String(encoder, msg)
	if err != nil {
		return fmt.Errorf("erro ao codificar a mensagem para Unicode: %v", err)
	}

	msgBox.SendKeys(encodedMsg)
	// Send the message by pressing Enter
	msgBox.SendKeys(selenium.EnterKey)

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
