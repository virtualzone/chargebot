'use client'

import { Container } from "react-bootstrap"
import { CopyBlock } from "react-code-blocks"

export default function PageHelp() {
  const shellCommandPushSurplus = `shell_command:
  push_pv_surplus: >
    curl --header 'Content-Type: application/json' --data '"surplus_watts": {{surplus}}}' http://localhost:8080/api/1/user/surplus`
  const haScriptSurplus = `service: shell_command.push_pv_surplus
  data:
    surplus: "{{ states('sensor.power_production_changeme') }}"`
  const shellCommandPlugState = `shell_command:
  push_tesla_plugged_in: >
    curl --header 'Content-Type: application/json' http://localhost:8080/api/1/user/{{vehicle}}/plugged_in
  push_tesla_unplugged: >
    curl --header 'Content-Type: application/json' http://localhost:8080/api/1/user/{{vehicle}}/unplugged`
  const haScriptPlugIn = `vehicle: your-vehicles-VIN`
  const dockerCompose = `services:
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
    data:`

  return (
    <Container fluid="sm" className="pt-5 container-max-width min-height">
      <h2 className="pb-3">Help</h2>

      <h5>How to get started?</h5>
      <ol>
        <li>Create an account at chargebot.io.</li>
        <li>Link your Tesla Account with chargebot.io and note down:
          <ul>
            <li>Your Tesla Token</li>
            <li> Your chargebot.io Token and Token Password</li>
          </ul>
        </li>
        <li>Set up your chargebot.io remote controller node using a <strong>docker-compose.yml</strong> file for Docker Compose:
        <CopyBlock text={dockerCompose} language="yaml" wrapLongLines={true} showLineNumbers={true} />
        </li>
        <li>Run using: <strong>docker compose up -d</strong></li>
        <li>Access the web frontend at: <a href="http://localhost:8080" target="_blank">http://localhost:8080</a></li>
      </ol>


      <h5 style={{ 'marginTop': '50px' }}>How does chargebot.io know about my solar surplus?</h5>
      <p>You'll need a home automation solution (such as Home Assistant, ioBroker or OpenHAB) or some other kind of scripting at your end which regularly pushes the available surplus to chargebot.io.</p>
      <p>Example for Home Assistant:</p>
      <ol>
        <li>Make sure Home Assistant knows about your surplus. This can i.e. be done by using a Shelly 3EM or a Tibber Pulse, which are integrated with your Home Assistant installation.</li>
        <li>
          Create a new shell command in your Home Assistant's <pre style={{ 'display': 'inline' }}>configuration.yaml</pre>:
          <CopyBlock text={shellCommandPushSurplus} language="yaml" wrapLongLines={true} showLineNumbers={true} />
        </li>
        <li>
          Restart Home Assistant.
        </li>
        <li>
          In Home Assistant, navigate to 'Settings' &gt; 'Automations &amp; scenes' &gt; 'Automations'.
        </li>
        <li>
          Create a new automation:
          <ul>
            <li>When (trigger): Time (i.e. every 5 minutes: Hours = *, Minutes = /5)</li>
            <li>Then do (action): Call service 'Shell Command: push_pv_surplus' with data:
              <CopyBlock text={haScriptSurplus} language="yaml" wrapLongLines={true} showLineNumbers={true} />
            </li>
          </ul>
        </li>
      </ol>

      <h5 style={{ 'marginTop': '50px' }}>How can I contribute?</h5>
      <p>Check out chargebot.io's <a href="https://github.com/virtualzone/chargebot" target="_blank">source code repository at GitHub</a>.</p>
    </Container>
  )
}
