# Rotina para envio de Mensagens de Whatsapp

Esse programa foi escrito em GO para envio de mensagens pelo Whatsapp com uma programação prévia em Json.

Utilize o compose.yaml para executar as funções.

# Instalar
Para instalar, a configuração mínima deverá ser uma máquina com 1Gb de RAM e processador com 2 cores. Abaixo disso, o sistema não consegue vincular o telefone ao Whatsapp Web do sistema.

Uma vez baixado o código do compose.yaml, você poderá usar o seguinte comando:

> docker-compose up -d

# Fazendo funcionar
Você deve escanear o QRCode com seu Whatsapp. Você terá 1 minutos para ver o log e escanear.

Para ver o qrcode, você deve executar o seguinte comando.

> docker-compose logs -f whatsapp

O log irá mostrar um QRCode em formato texto, conforme mostra a imagem abaixo.

![Código QRCode a ser escaneado ao ver os logs](./img/qrcode.png)

Escaneie e em seguida, será avisado sobre o término do tempo de escaneio.

# Mensagens
Para ver como formatar as mensagens. Veja o exemplo do arquivo [mensagens.json](data/mensagens.json)

É importante lembrar das seguintes variáveis:
- id - somente identificação. Definir seriado 1, 2, 3...
- destinatario - Nome do destinatário. Não usar número. O destinatário deverá estar em seus contatos no Whatsapp.
- conteudos - Mensagens escolhidas de forma aleatória
- ultimo_envio - Não precisa definir nada. Atualizada de acordo com o uso
- horario_envio - Horário em que a mensagem será enviada. Formato 24:00
- dia_semana - Dias que as mensagens serão enviadas. 0 (domingo) a 6 (sábado).

