# Rotina para envio de Mensagens de Whatsapp

Esse programa foi escrito em GO para envio de mensagens pelo Whatsapp com uma programação prévia em Json.

Utilize o compose.yaml para executar as funções.

# Instalar
> docker run -d --name whatsapp -v ./data:/app/data:rw quay.io/dioclecio/whatsapp-cron:latest

# Fazendo funcionar
Em seguida você deve escanear o QRCode com seu Whatsapp.
Para isso, você terá 1 minutos para ver o log e escanear.

> docker logs whatsapp 

# Mensagens
Para ver como formatar as mensagens. Veja o exemplo do arquivo ![data/mensagens.json](data/mensagens.json)

