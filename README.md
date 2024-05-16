# chargebot.io
chargebot.io allows for charging your Tesla from your solar power plant and/or at the lowest grid prices (i.e. when using Tibber). Works with any wallbox and any inverter.

### Features
* Controls a Tesla's charging process (start, stop, amps) via Tesla's new Fleet API
* Supports charging on solar surplus and/or when dynamic grid prices are at their lowest
* Queries Tibber's API to retrieve upcoming grid prices
* Gets input regarding your current solar surplus via REST API push of by subscribing to an MQTT topic
* Easy to use web frontend for setting parameters and checking your vehicle's charging process
* Freely settable options for minimum surplus, minimum charge time, surplus buffer and more 
* Hosted locally in your own network using Docker

## Get started
1. Create an account at: https://chargebot.io
1. Link your Tesla Account with chargebot.io and note down:
   * Your Tesla Token
   * Your chargebot.io Token and Token Password
1. Set up your chargebot.io remote controller node using a ```docker-compose.yml``` file for Docker Compose:
   ```
   services:
     node:
       image: ghcr.io/virtualzone/chargebot:latest
       restart: always
       ports:
         - 8080:8080
       environment:
         TESLA_REFRESH_TOKEN: 'initial-tesla-refresh-token'
         DB_FILE: '/data/chargbeot.db'
         PORT: '8080'
         TOKEN: 'your-chargebot.io-token'
         PASSWORD: 'your-chargebot-io-token-password'
         CRYPT_KEY: 'a-32-bytes-long-random-key'
       volumes:
         - db:/data
     volumes: 
       data:
   ```
1. Run using: ```docker compose up -d```
1. Access the web frontend at: http://localhost:8080

## How it works
chargebot.io uses the Tesla Fleet API and Tesla Fleet Telemetry in order to control your vehicle's charging process.

The actual work is done by your local remote controller node. It decides whether there's enough surplus from your solar power plant in order to charge your Tesla. It checks your grid provider for the current prices and starts charging if the prices are below your defined maximum.

The centralized chargebot.io instance serves as a proxy for your local node's command and forwards them to the Tesla Fleet API. The centralized instance is required as it signs requests from your local node to your Tesla with a private key and forwards incoming Fleet Telemetry data to your local node.

Only your local node knows and saves your personal Tesla Token. It is neither stored nor used by the centralized chargebot.io instance.

## Push notifications
chargebot.io supports sending push notifications using Telegram. To set it up, follow these steps:

1. Create a bot by sending ```/newbot``` to Telegram's [@BotFather](https://t.me/BotFather) by following [these instructions](https://core.telegram.org/bots/features#botfather) and note down the displayed token.
1. Find out your Telegram User ID by i.e. sending any message to the [GetIDs bot](https://t.me/getidsbot).
1. Set the ```TELEGRAM_TOKEN``` and ```TELEGRAM_CHAT_ID``` environment variables and restart your node.

## Environment variables
| Environment Variable | Type | Default | Description |
| --- | --- | --- | --- |
| TESLA_REFRESH_TOKEN | string |  | Tesla Refresh Token shown after linking your Tesla Account with your chargebot.io account (only needed for initial setup) |
| DB_FILE | string | /tmp/chargebot_node.db | SQLite database file |
| PORT | int | 8080 | HTTP listening port |
| TOKEN | string |  | Your chargebot.io token |
| PASSWORD | string |  | Your chargebot.io token's password |
| CRYPT_KEY | string |  | A key for encrypting your Tesla Refresh Token in the SQLite database |
| TELEGRAM_TOKEN | string |  | Telegram Bot Authentication Token for push notifications |
| TELEGRAM_CHAT_ID | string |  | Telegram Chat ID for push notifications |
| PLUG_AUTODETECT | bool | 1 | Automatically detect vehicle's plugged in state (else, use the webhooks to notify node about plugged state) |
| MQTT_BROKER | string | | MQTT Broker address (i.e. 'tcp://broker.hivemq.com:1883') |
| MQTT_CLIENT_ID | string | chargebot | MQTT Client ID |
| MQTT_USERNAME | string | | MQTT username |
| MQTT_PASSWORD | string | | MQTT password |
| MQTT_TOPIC_SURPLUS | string | chargebot/surplus | MQTT topic for solar surplus |

## More help
Visit https://chargebot.io/help/ for more information.