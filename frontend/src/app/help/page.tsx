'use client'

import { Container } from "react-bootstrap"
import { CopyBlock } from "react-code-blocks"

export default function PageHelp() {
  const shellCommandPushSurplus = `shell_command:
  push_pv_surplus: >
    curl --header 'Content-Type: application/json' --data '{"password": "{{password}}", "surplus_watts": {{surplus}}}' https://chargebot.io/api/1/user/{{token}}/surplus`
  const haScriptSurplus = `service: shell_command.push_pv_surplus
  data:
    token: your-chargebot.io-token
    password: your-chargebot.io-password
    surplus: "{{ states('sensor.power_production_changeme') }}"`
  const shellCommandPlugState = `shell_command:
  push_tesla_plugged_in: >
    curl --header 'Content-Type: application/json' --data '{"password": "{{password}}"}' https://chargebot.io/api/1/user/{{token}}/plugged_in
  push_tesla_unplugged: >
    curl --header 'Content-Type: application/json' --data '{"password": "{{password}}"}' https://chargebot.io/api/1/user/{{token}}/unplugged`
  const haScriptPlugIn = `token: your-chargebot.io-token\npassword: your-chargebot.io-password`

  return (
    <Container fluid="sm" className="pt-5 container-max-width min-height">
      <h2 className="pb-3">Help</h2>

      <h5>How does chargebot.io know about my solar surplus?</h5>
      <p>At the moment, you'll need a home automation solution (such as Home Assistant, ioBroker or OpenHAB) or some other kind of scripting at your end which regularly pushes the available surplus to chargebot.io.</p>
      <p>Example for Home Assistant:</p>
      <ol>
        <li>Make sure Home Assistant knows about your surplus. This can i.e. be done by using a Shelly 3EM or a Tibber Pulse, which are integrated with your Home Assistant installation.</li>
        <li>
          Create a new shell command in your Home Assistant's <pre style={{'display': 'inline'}}>configuration.yaml</pre>:
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

      <h5 style={{'marginTop': '50px'}}>How does chargebot.io know if my Tesla is connected to my home charger?</h5>
      <p>At the moment, you'll need a home automation solution (such as Home Assistant, ioBroker or OpenHAB) or some other kind of scripting at your end which sends your Tesla's plugged in / plugged out state to chargebot.io.</p>
      <p>Example for Home Assistant:</p>
      <ol>
        <li>Make sure Home Assistant knows if your Tesla is connected. For a Tesla Wall Connector (TWC), this can be achieved by setting up the <a href="https://www.home-assistant.io/integrations/tesla_wall_connector/" target="_blank">Tesla Wall Connector integration</a>.</li>
        <li>
          Create a new shell command in your Home Assistant's <pre style={{'display': 'inline'}}>configuration.yaml</pre>:
          <CopyBlock text={shellCommandPlugState} language="yaml" wrapLongLines={true} showLineNumbers={true} />
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
            <li>When (trigger): Entity 'Tesla Wall Connector Vehicle connected' changes to 'Plugged in'</li>
            <li>
              Then do (action): Call service 'Shell Command: push_tesla_plugged_in' with data:
              <CopyBlock text={haScriptPlugIn} language="yaml" wrapLongLines={true} showLineNumbers={true} />
            </li>
          </ul>
        </li>
        <li>
          Create a second automation:
          <ul>
            <li>When (trigger): Entity 'Tesla Wall Connector Vehicle connected' changes to 'Unplugged'</li>
            <li>
              Then do (action): Call service 'Shell Command: push_tesla_plugged_unplugged' with data:
              <CopyBlock text={haScriptPlugIn} language="yaml" wrapLongLines={true} showLineNumbers={true} />
            </li>
          </ul>
        </li>
      </ol>
    </Container>
  )
}
