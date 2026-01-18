# Simple port forwarding

Консольное приложение предназначено для перенаправления tcp-соединений.
Создавалось для перенаправления RDP-соединений. Управляющий порт, которое служает приложение 80, задаётся в файле config.json.

# Настройка

В каталоге с утилитой должен лежать файл настроек `config.json`

*Пример*:

```json
{
	"localPort": 80,
	"remoteHosts": [
		{
			"id": "498117ae-079d-42ae-845e-005a36012858",
			"host": "host001",
			"port": 3989
		},
		{
			"id": "498117ae-079d-42ae-845e-005a3601285",
			"host": "host002",
			"port": 3989
		}
	]
}
```

# Подключение к RDP нужно сервера через файл bat

*Пример*:

```bat
@echo off
setlocal enabledelayedexpansion

set URL=http://host001
set ID=498117ae-079d-42ae-845e-005a36012858

for /f %%a in ('curl -s -o nul -w "%%{http_code}" "%URL%?id=%ID%" 2^>^&1') do set HTTP_CODE=%%a

if "!HTTP_CODE!"=="200" (
    echo Success! Starting RDP session...
    start mstsc /v:rdssc04srvclsb2:80
) else (
    echo Error: Received HTTP code !HTTP_CODE!
)

endlocal
```

# Логирование
Логи сервиса находятся в каталоге "log" в папке с исполняемым файлом.

