'use client'

import { checkAuth, deleteAPI, getAPI, postAPI, putAPI } from "../util";
import { useEffect, useState } from "react";
import Loading from "../loading";
import { Accordion, Button, Container, Form, InputGroup, Modal, Table } from "react-bootstrap";
import { CopyBlock } from "react-code-blocks";
import { useRouter } from "next/navigation";
import { Loader as IconLoad } from 'react-feather';
import Link from "next/link";

export default function Authorized() {
  let vehicleID = 0
  const router = useRouter()
  const [vehicle, setVehicle] = useState({} as any)
  const [isLoading, setLoading] = useState(true)
  const [savingVehicle, setSavingVehicle] = useState(false)
  const [savingApiToken, setSavingApiToken] = useState(false)
  const [chargingEnabled, setChargingEnabled] = useState(false)
  const [targetSoC, setTargetSoC] = useState(0)
  const [maxAmps, setMaxAmps] = useState(0)
  const [numPhases, setNumPhases] = useState(0)
  const [chargeOnSurplus, setChargeOnSurplus] = useState(false)
  const [minSurplus, setMinSurplus] = useState(0)
  const [minChargetime, setMinChargetime] = useState(0)
  const [chargeOnTibber, setChargeOnTibber] = useState(false)
  const [gridProvider, setGridProvider] = useState('tibber')
  const [gridStrategy, setGridStrategy] = useState(1)
  const [maxPrice, setMaxPrice] = useState(0)
  const [departDays, setDepartDays] = useState([1, 2, 3, 4, 5])
  const [departTime, setDepartTime] = useState('07:00')
  const [tibberToken, setTibberToken] = useState('')
  const [showTokenHelp, setShowTokenHelp] = useState(false)
  const [vehicleState, setVehicleState] = useState({} as any)
  const [surpluses, setSurpluses] = useState([] as any)
  const [chargingEvents, setChargingEvents] = useState([] as any)
  const [manCtrlLimit, setManCtrlLimit] = useState(50)
  const [manCtrlAmps, setManCtrlAmps] = useState(16)
  const [manCtrlEnabled, setManCtrlEnabled] = useState(0)
  const [manCtrlMins, setManCtrlMins] = useState(0)

  useEffect(() => {
    const searchParams = new URLSearchParams(window.location.search);
    const id = searchParams.get("id");
    if (!id || window.isNaN(Number(id))) {
      router.push('/authorized')
      return;
    }
    vehicleID = Number(id);
    const fetchData = async () => {
      await checkAuth();
      await loadVehicle();
      setLoading(false);
    }
    fetchData();
  }, [router]);

  const loadVehicleState = async (token: string) => {
    if (!token) {
      return
    }
    const json = await getAPI("/api/1/user/" + token + "/state");
    if (json) {
      setVehicleState(json);
    }
  }

  const loadLatestSurpluses = async (token: string) => {
    if (!token) {
      return
    }
    const json = await getAPI("/api/1/user/" + token + "/surplus");
    setSurpluses(json);
  }

  const loadLatestChargingEvents = async (token: string) => {
    if (!token) {
      return
    }
    const json = await getAPI("/api/1/user/" + token + "/events");
    setChargingEvents(json);
  }

  const loadVehicle = async () => {
    const json = await getAPI("/api/1/tesla/my_vehicles");
    (json as any[]).forEach(e => {
      if (e.id === vehicleID) {
        setChargingEnabled(e.enabled);
        setTargetSoC(e.target_soc);
        setMaxAmps(e.max_amps);
        setNumPhases(e.num_phases);
        setChargeOnSurplus(e.surplus_charging);
        setMinSurplus(e.min_surplus);
        setMinChargetime(e.min_chargetime);
        setChargeOnTibber(e.lowcost_charging);
        setMaxPrice(e.max_price);
        setGridProvider(e.gridProvider);
        setGridStrategy(e.gridStrategy);
        setDepartDays([...e.departDays].map(i => Number(i)));
        setDepartTime(e.departTime);
        setTibberToken(e.tibber_token);
        setShowTokenHelp(false);
        loadVehicleState(e.api_token);
        loadLatestSurpluses(e.api_token);
        loadLatestChargingEvents(e.api_token);
        setVehicle(e);
      }
    });
  }

  function generateAPIToken() {
    const fetchData = async () => {
      setSavingApiToken(true);
      const json = await postAPI("/api/1/tesla/api_token_create", { vehicle_id: vehicle.id });
      if (vehicle) {
        let vehicleNew = {
          ...vehicle,
          api_token: json.token,
          api_password: json.password,
        };
        setVehicle(vehicleNew);
      }
      setSavingApiToken(false);
    }
    fetchData();
  }

  function updateAPITokenPassword(id: string) {
    const fetchData = async () => {
      setSavingApiToken(true);
      const json = await postAPI("/api/1/tesla/api_token_update/" + id, {});
      if (vehicle) {
        let vehicleNew = {
          ...vehicle,
          api_token: json.token,
          api_password: json.password,
        };
        setVehicle(vehicleNew);
      }
      setSavingApiToken(false);
    }
    fetchData();
  }

  function saveVehicle() {
    const fetchData = async () => {
      setSavingVehicle(true);
      let payload = {
        "enabled": chargingEnabled,
        "target_soc": targetSoC,
        "max_amps": maxAmps,
        "num_phases": numPhases,
        "surplus_charging": chargeOnSurplus,
        "min_surplus": minSurplus,
        "min_chargetime": minChargetime,
        "lowcost_charging": chargeOnTibber,
        "gridProvider": gridProvider,
        "gridStrategy": gridStrategy,
        "departDays": departDays.join(''),
        "departTime": departTime,
        "max_price": maxPrice,
        "tibber_token": tibberToken
      };
      await putAPI("/api/1/tesla/vehicle_update/" + vehicle.id, payload);
      await loadVehicle();
      setSavingVehicle(false);
    }
    fetchData();
  }

  function deleteVehicle() {
    if (!window.confirm("Delete his vehicle?")) {
      return;
    }
    const fetchData = async () => {
      setLoading(true);
      await deleteAPI("/api/1/tesla/vehicle_delete/" + vehicle.id);
      router.push('/authorized/?removed=1');
    }
    fetchData();
  }

  function getChargeStateText(id: number) {
    if (id === 0) return 'Not charging';
    if (id === 1) return 'Charging on solar';
    if (id === 2) return 'Charging on grid';
    return 'Unknown';
  }

  function getChargingEventText(id: number) {
    if (id === 1) return 'Charging started';
    if (id === 2) return 'Charging stopped';
    if (id === 3) return 'Vehicle plugged in';
    if (id === 4) return 'Vehicle unplugged';
    if (id === 5) return 'Updated vehicle data';
    if (id === 6) return 'Wake vehicle';
    if (id === 7) return 'Set target SoC';
    if (id === 8) return 'Set charge amps';
    if (id === 9) return 'Set scheduled charging';
    return 'Unknown';
  }

  async function copyToClipboard(s: string) {
    try {
      await navigator.clipboard.writeText(s);
    } catch (err) {
      console.error('Failed to copy: ', err);
    }
  }

  function getMaxChargingPower() {
    let i = 0;
    if (maxAmps !== undefined && maxAmps !== undefined) {
      i = maxAmps!;
    }
    let phases = 0;
    if (numPhases !== undefined && numPhases !== undefined) {
      phases = numPhases!;
    }
    let p = i * phases * 230;
    if (p > 1000) {
      return Math.round(p / 1000) + " kW";
    }
    return p + " W";
  }

  function manualControlTestDrive() {
    postAPI("/api/1/ctrl/" + vehicle.id + "/testDrive", null).then(res => {
      window.alert('Charging should start shortly. It will be stopped after 30 seconds. Please check your Tesla App if this automation works.');
    })
  }

  if (isLoading) {
    return <Loading />
  }

  let code1 = "curl --header 'Content-Type: application/json' --data '{\"password\": \"\", \"surplus_watts\": 1500}' https://chargebot.io/api/1/user/" + vehicle.api_token + "/surplus";
  let code2 = "curl --header 'Content-Type: application/json' --data '{\"password\": \"\", \"inverter_active_power_watts\": 2000, \"consumption_watts\": 200}' https://chargebot.io/api/1/user/" + vehicle.api_token + "/surplus";
  let code3 = "curl --header 'Content-Type: application/json' --data '{\"password\": \"\"}' https://chargebot.io/api/1/user/" + vehicle.api_token + "/plugged_in";
  let code4 = "curl --header 'Content-Type: application/json' --data '{\"password\": \"\"}' https://chargebot.io/api/1/user/" + vehicle.api_token + "/unplugged";
  let tokenHelp = (
    <Modal show={showTokenHelp} onHide={() => setShowTokenHelp(false)}>
      <Modal.Header closeButton>
        <Modal.Title>How to use your API Token</Modal.Title>
      </Modal.Header>

      <Modal.Body>
        <h5>Update surplus</h5>
        <p>Regularly push your enegery surplus available for charging your vehicle (inverter active power minus consumption) using HTTP POST:</p>
        <CopyBlock text={code1} language="bash" wrapLongLines={true} showLineNumbers={false} />
        <p>Alternatively, you can push your current inverter active power and your household&apos;s consumption separately:</p>
        <CopyBlock text={code2} language="bash" wrapLongLines={true} showLineNumbers={false} />
        <h5>Update plugged in status</h5>
        <p>If your vehicles gets plugged in:</p>
        <CopyBlock text={code3} language="bash" wrapLongLines={true} showLineNumbers={false} />
        <p>If your vehicles gets unplugged:</p>
        <CopyBlock text={code4} language="bash" wrapLongLines={true} showLineNumbers={false} />
      </Modal.Body>
    </Modal>
  );

  let surplusRows = <tr><td colSpan={2}>No records founds</td></tr>;
  if (surpluses && surpluses.length > 0) {
    surplusRows = surpluses.map((s: any) => {
      return (
        <tr key={"surplus-" + s.ts}>
          <td>{s.ts.replace('T', ' ').replace('Z', '')}</td>
          <td>{s.surplus_watts} W</td>
        </tr>
      );
    });
  }
  let surplusTable = (
    <Table>
      <thead>
        <tr>
          <th>Time (UTC)</th>
          <th>Surplus</th>
        </tr>
      </thead>
      <tbody>
        {surplusRows}
      </tbody>
    </Table>
  );

  let eventRows = <tr><td colSpan={3}>No records founds</td></tr>;
  if (chargingEvents && chargingEvents.length > 0) {
    eventRows = chargingEvents.map((s: any) => {
      return (
        <tr key={"event-" + s.ts}>
          <td>{s.ts.replace('T', ' ').replace('Z', '')}</td>
          <td>{getChargingEventText(s.event)}</td>
          <td>{s.data}</td>
        </tr>
      );
    });
  }
  let eventsTable = (
    <Table>
      <thead>
        <tr>
          <th>Time (UTC)</th>
          <th>Event</th>
          <th>Details</th>
        </tr>
      </thead>
      <tbody>
        {eventRows}
      </tbody>
    </Table>
  );

  let token = <Button variant="primary" onClick={() => generateAPIToken()} disabled={savingApiToken}>{savingApiToken ? <><IconLoad className="feather-button loader" /> Creating token...</> : 'Create API Token'}</Button>
  if (vehicle.api_token) {
    token = <>
      <strong>API Token:</strong>
      <Button variant="link" onClick={() => copyToClipboard(vehicle.api_token)}>Copy</Button>
      <br />
      <pre>{vehicle.api_token}</pre>
      <strong>Password:</strong>
      <Button variant="link" onClick={() => updateAPITokenPassword(vehicle.api_token)} disabled={savingApiToken}>{savingApiToken ? <><IconLoad className="feather-button loader" /> Updating...</> : 'Update'}</Button>
      <br />
      <pre>****************</pre>
    </>
    if (vehicle.api_password) {
      token = <>
        <strong>API Token:</strong>
        <Button variant="link" onClick={() => copyToClipboard(vehicle.api_token)}>Copy</Button>
        <br />
        <pre>{vehicle.api_token}</pre>
        <strong>Password:</strong>
        <Button variant="link" onClick={() => copyToClipboard(vehicle.api_password)}>Copy</Button>
        <br />
        <pre>{vehicle.api_password}</pre>
      </>
    }
  }
  let chargePrefs = (
    <Form onSubmit={e => { e.preventDefault(); e.stopPropagation(); saveVehicle() }}>
      <Form.Check // prettier-ignore
        type="switch"
        label="Enable smart charging control"
        checked={chargingEnabled}
        onChange={e => setChargingEnabled(e.target.checked)}
      />
      <InputGroup className="mb-3">
        <Form.Control
          placeholder="Target SoC"
          aria-label="Target SoC"
          aria-describedby="target-soc-addon1"
          type="number"
          min={1}
          max={100}
          required={chargingEnabled}
          disabled={!chargingEnabled}
          value={targetSoC}
          onChange={e => setTargetSoC(Number(e.target.value))}
        />
        <InputGroup.Text id="target-soc-addon1">%</InputGroup.Text>
      </InputGroup>
      <InputGroup className="mb-3">
        <Form.Control
          placeholder="Max. Amps"
          aria-label="Max. Amps"
          aria-describedby="maxamps-addon1"
          type="number"
          min={1}
          max={32}
          required={chargingEnabled}
          disabled={!chargingEnabled}
          value={maxAmps}
          onChange={e => setMaxAmps(Number(e.target.value))}
        />
        <InputGroup.Text id="maxamps-addon1">A</InputGroup.Text>
      </InputGroup>
      <InputGroup className="mb-3">
        <Form.Select
          aria-label="Number of Phases"
          required={chargingEnabled}
          disabled={!chargingEnabled}
          value={numPhases}
          onChange={e => setNumPhases(Number(e.target.value))}>
          <option value="1">uniphase</option>
          <option value="3">three-phase</option>
        </Form.Select>
      </InputGroup>
      <InputGroup className="mb-3">
        <Form.Control plaintext={true} readOnly={true} defaultValue={'Up to ' + getMaxChargingPower()} />
      </InputGroup>
      <Form.Check // prettier-ignore
        type="switch"
        label="Charge on surplus of solar energy"
        checked={chargeOnSurplus}
        onChange={e => setChargeOnSurplus(e.target.checked)}
        style={{ 'marginTop': '25px' }}
      />
      <InputGroup className="mb-3">
        <Form.Control
          placeholder="Minimum surplus"
          aria-label="Minimum surplus"
          aria-describedby="min-surplus-addon1"
          type="number"
          min={1}
          max={10000}
          required={chargeOnSurplus}
          disabled={!chargeOnSurplus}
          value={minSurplus}
          onChange={e => setMinSurplus(Number(e.target.value))}
        />
        <InputGroup.Text id="min-surplus-addon1">Watts</InputGroup.Text>
      </InputGroup>
      <InputGroup className="mb-3">
        <Form.Control
          placeholder="Minimum Charging Time"
          aria-label="Minimum Charging Time"
          aria-describedby="chargetime-addon1"
          type="number"
          min={1}
          max={120}
          required={chargeOnSurplus}
          disabled={!chargeOnSurplus}
          value={minChargetime}
          onChange={e => setMinChargetime(Number(e.target.value))}
        />
        <InputGroup.Text id="chargetime-addon1">Minutes</InputGroup.Text>
      </InputGroup>
      <Form.Check // prettier-ignore
        type="switch"
        label="Charge on low grid price"
        checked={chargeOnTibber}
        onChange={e => setChargeOnTibber(e.target.checked)}
        style={{ 'marginTop': '25px' }}
      />
      <InputGroup className="mb-3">
        <Form.Select
          aria-label="Provider"
          required={chargeOnTibber}
          disabled={!chargeOnTibber}
          value={gridProvider}
          onChange={e => setGridProvider(e.target.value)}>
          <option value="tibber">Tibber</option>
        </Form.Select>
      </InputGroup>
      <InputGroup className="mb-3">
        <Form.Select
          aria-label="Strategy"
          required={chargeOnTibber}
          disabled={!chargeOnTibber}
          value={gridStrategy}
          onChange={e => setGridStrategy(Number(e.target.value))}>
          <option value="1">min. price, but at least below x</option>
          <option value="2">price below x, possibly charged at departure</option>
          <option value="3">min. price, certainly charged at departure</option>
        </Form.Select>
      </InputGroup>
      <InputGroup className="mb-3" hidden={gridStrategy === 3}>
        <Form.Control
          placeholder="Maximum Grid Price"
          aria-label="Maximum Grid Price"
          aria-describedby="tibber-price-addon1"
          type="number"
          min={1}
          max={100}
          required={chargeOnTibber}
          disabled={!chargeOnTibber}
          value={maxPrice}
          onChange={e => setMaxPrice(Number(e.target.value))}
        />
        <InputGroup.Text id="tibber-price-addon1">Cents</InputGroup.Text>
      </InputGroup>
      <InputGroup className="mb-3" hidden={gridStrategy === 1}>
        <Form.Check inline={true} label="Mo" type="checkbox" disabled={!chargeOnTibber} checked={departDays.indexOf(1) > -1} onChange={e => setDepartDays(e.target.checked ? [...departDays, 1] : [...departDays].toSpliced(departDays.indexOf(1), 1))} />
        <Form.Check inline={true} label="Tu" type="checkbox" disabled={!chargeOnTibber} checked={departDays.indexOf(2) > -1} onChange={e => setDepartDays(e.target.checked ? [...departDays, 2] : [...departDays].toSpliced(departDays.indexOf(2), 1))} />
        <Form.Check inline={true} label="We" type="checkbox" disabled={!chargeOnTibber} checked={departDays.indexOf(3) > -1} onChange={e => setDepartDays(e.target.checked ? [...departDays, 3] : [...departDays].toSpliced(departDays.indexOf(3), 1))} />
        <Form.Check inline={true} label="Th" type="checkbox" disabled={!chargeOnTibber} checked={departDays.indexOf(4) > -1} onChange={e => setDepartDays(e.target.checked ? [...departDays, 4] : [...departDays].toSpliced(departDays.indexOf(4), 1))} />
        <Form.Check inline={true} label="Fr" type="checkbox" disabled={!chargeOnTibber} checked={departDays.indexOf(5) > -1} onChange={e => setDepartDays(e.target.checked ? [...departDays, 5] : [...departDays].toSpliced(departDays.indexOf(5), 1))} />
        <Form.Check inline={true} label="Sa" type="checkbox" disabled={!chargeOnTibber} checked={departDays.indexOf(6) > -1} onChange={e => setDepartDays(e.target.checked ? [...departDays, 6] : [...departDays].toSpliced(departDays.indexOf(6), 1))} />
        <Form.Check inline={true} label="Su" type="checkbox" disabled={!chargeOnTibber} checked={departDays.indexOf(7) > -1} onChange={e => setDepartDays(e.target.checked ? [...departDays, 7] : [...departDays].toSpliced(departDays.indexOf(7), 1))} />
      </InputGroup>
      <InputGroup className="mb-3" hidden={gridStrategy === 1}>
        <Form.Control
          type="time"
          min="00:00"
          max="23:59"
          required={chargeOnTibber}
          disabled={!chargeOnTibber}
          value={departTime}
          onChange={e => setDepartTime(e.target.value)}
        />
      </InputGroup>
      <InputGroup className="mb-3" hidden={gridProvider !== 'tibber'}>
        <Form.Control
          placeholder="Tibber Token"
          aria-label="Tibber Token"
          type="text"
          required={chargeOnTibber}
          disabled={!chargeOnTibber}
          value={tibberToken}
          onChange={e => setTibberToken(e.target.value)}
        />
      </InputGroup>
      <p><a href="https://developer.tibber.com/settings/accesstoken" target="_blank">Get your Tibber Access Token</a></p>
      <Button type="submit" variant="primary" disabled={savingVehicle}>{savingVehicle ? <><IconLoad className="feather-button loader" /> Saving...</> : 'Save'}</Button>
    </Form>
  );
  let accordionSurpluses = <></>;
  if (vehicle.api_token) {
    accordionSurpluses = (
      <Accordion.Item eventKey="3">
        <Accordion.Header>Latest recorded surpluses</Accordion.Header>
        <Accordion.Body>
          {surplusTable}
        </Accordion.Body>
      </Accordion.Item>
    );
  }
  let accordionChargingEvents = <></>;
  if (vehicle.api_token) {
    accordionChargingEvents = (
      <Accordion.Item eventKey="4">
        <Accordion.Header>Latest charging events</Accordion.Header>
        <Accordion.Body>
          {eventsTable}
        </Accordion.Body>
      </Accordion.Item>
    );
  }
  let accordionState = <></>;
  if ((vehicle.api_token) && (vehicleState)) {
    accordionState = (
      <Accordion.Item eventKey="2">
        <Accordion.Header>Vehicle State</Accordion.Header>
        <Accordion.Body>
          <Table>
            <tbody>
              <tr>
                <td>Plugged In</td>
                <td>{vehicleState.pluggedIn ? 'Yes' : 'No'}</td>
              </tr>
              <tr>
                <td>Charging State</td>
                <td>{getChargeStateText(vehicleState.chargingState)}</td>
              </tr>
              <tr>
                <td>SoC</td>
                <td>{vehicleState.soc} %</td>
              </tr>
              <tr>
                <td>Amps</td>
                <td>{vehicleState.amps} A</td>
              </tr>
            </tbody>
          </Table>
        </Accordion.Body>
      </Accordion.Item>
    );
  }
  let accordionManualControl = (
    <Accordion.Item eventKey="5">
      <Accordion.Header>Test Drive</Accordion.Header>
      <Accordion.Body>
        <p>You can check if chargebot.io can control your vehicle's charging process.</p>
        <p>After clicking the button below, your vehicle should...</p>
        <ul>
          <li>wake up (if asleep),</li>
          <li>start the charging process,</li>
          <li>wait for 30 seconds,</li>
          <li>then stop the charging process.</li>
        </ul>
        <p>
          <Button variant="primary" onClick={() => manualControlTestDrive()}>Start charge test</Button>
        </p>
      </Accordion.Body>
    </Accordion.Item>
  );
  return (
    <Container fluid="sm" className="pt-5 container-max-width min-height">
      <h2 className="pb-3">{vehicle.display_name}</h2>
      <p>{vehicle.vin}</p>
      <p>Before chargebot.io can control your vehicle's charging process, you need to set up the virtual key:</p>
      <p><a href="https://tesla.com/_ak/chargebot.io" target="_blank">Set Up Virtual Key</a></p>
      <br />
      <Accordion defaultActiveKey="0" flush={true}>
        <Accordion.Item eventKey="0">
          <Accordion.Header>Charging Preferences</Accordion.Header>
          <Accordion.Body>
            {chargePrefs}
          </Accordion.Body>
        </Accordion.Item>
        <Accordion.Item eventKey="1">
          <Accordion.Header>API Token</Accordion.Header>
          <Accordion.Body>
            <p>An individual API token is required to send your solar surplus and plug in events to chargebot.io.</p>
            <p>{token}</p>
            <div hidden={vehicle.api_token === ''}>
              {tokenHelp}
              <Button variant="primary" onClick={e => setShowTokenHelp(true)}>How to use</Button>
            </div>
          </Accordion.Body>
        </Accordion.Item>
        {accordionState}
        {accordionSurpluses}
        {accordionChargingEvents}
        {accordionManualControl}
        <Accordion.Item eventKey="99">
          <Accordion.Header>Danger zone</Accordion.Header>
          <Accordion.Body>
            <Button variant="danger" onClick={() => deleteVehicle()}>Remove vehicle from chargebot.io</Button>
          </Accordion.Body>
        </Accordion.Item>
      </Accordion>
      <Link href="/authorized" className="btn btn-link">&lt; Back</Link>
    </Container>
  );
}
