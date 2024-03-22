# chargebot.io
chargebot.io allows for charging your Tesla from your solar power plant and/or at the lowest grid prices (i.e. when using Tibber). Works with any wallbox and any inverter.

## Get started
1. Create an account at: https://chargebot.io
1. Link your Tesla Account with chargebot.io and note down:
   * Your Tesla Token
   * Your chargebot.io Token and Token Password
1. Set up your chargebot.io remote controller node using Docker:
   ```
   docker run 
   ```
1. Open 

## How it works
chargebot.io uses the Tesla Fleet API and Tesla Fleet Telemetry in order to control your vehicle's charging process.

The actual work is done by your local remote controller node. It decides whether there's enure surplus from your solar power plant in order to charge your Tesla. It checks your grid provider for the current prices and starts charging if the prices are below your defined maximum.

The centralized chargebot.io instance serves as a proxy for your local node's command and forwards them to the Tesla Fleet API. The centralized instance is required as it signs requests from your local node to your Tesla with a private key and forwards incoming Fleet Telemetry data to your local node.

Only your local node knows and saved your personal Tesla Token. It is neither stored nor used by the centralized chargebot.io instance.