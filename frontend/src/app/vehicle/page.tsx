'use client'

import { checkAuth, deleteAPI, getAPI, postAPI, putAPI } from "../util";
import { useEffect, useState } from "react";
import Loading from "../loading";
import { Accordion, Button, Container, Form, InputGroup, Modal, Table } from "react-bootstrap";
import { CopyBlock } from "react-code-blocks";
import { useRouter } from "next/navigation";
import Link from "next/link";

export default function Authorized() {
  let vehicleID = 0
  const router = useRouter()
  const [vehicle, setVehicle] = useState({} as any)
  const [isLoading, setLoading] = useState(true)
  const [chargingEnabled, setChargingEnabled] = useState(false)
  const [targetSoC, setTargetSoC] = useState(0)
  const [maxAmps, setMaxAmps] = useState(0)
  const [numPhases, setNumPhases] = useState(0)
  const [chargeOnSurplus, setChargeOnSurplus] = useState(false)
  const [minSurplus, setMinSurplus] = useState(0)
  const [minChargetime, setMinChargetime] = useState(0)
  const [chargeOnTibber, setChargeOnTibber] = useState(false)
  const [maxPrice, setMaxPrice] = useState(0)
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
      const json = await postAPI("/api/1/tesla/api_token_create", { vehicle_id: vehicle.id });
      if (vehicle) {
        vehicle.api_token = json.token;
        vehicle.api_password = json.password;
      }
      setVehicle(vehicle);
    }
    fetchData();
  }

  function updateAPITokenPassword(id: string) {
    const fetchData = async () => {
      const json = await postAPI("/api/1/tesla/api_token_update/" + id, {});
      if (vehicle) {
        vehicle.api_token = json.token;
        vehicle.api_password = json.password;
      }
      setVehicle(vehicle);
    }
    fetchData();
  }

  function saveVehicle() {
    const fetchData = async () => {
      setLoading(true);
      let payload = {
        "enabled": chargingEnabled,
        "target_soc": targetSoC,
        "max_amps": maxAmps,
        "num_phases": numPhases,
        "surplus_charging": chargeOnSurplus,
        "min_surplus": minSurplus,
        "min_chargetime": minChargetime,
        "lowcost_charging": chargeOnTibber,
        "max_price": maxPrice,
        "tibber_token": tibberToken
      };
      await putAPI("/api/1/tesla/vehicle_update/" + vehicle.id, payload);
      await loadVehicle();
      setLoading(false);
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
      router.push('/authorized');
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

  function manualControl(api: string, i1: number, i2: number) {
    let s = "";
    if (i1 >= 0) {
      s = "/" + i1;
    }
    if (i2 >= 0) {
      s += "/" + i2;
    }
    postAPI("/api/1/ctrl/" + vehicle.id + "/" + api + s, null).then(res => {
      window.alert(JSON.stringify(res));
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

  let token = <Button variant="link" onClick={() => generateAPIToken()}>Generate API Token</Button>
  if (vehicle.api_token) {
    token = <>
      API Token: {vehicle.api_token}
      <br />
      Password: ****************
      <Button variant="link" onClick={() => updateAPITokenPassword(vehicle.api_token)}>Update</Button>
    </>
    if (vehicle.api_password) {
      token = <>API Token: {vehicle.api_token}<br /> Password: {vehicle.api_password}</>
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
      <Form.Select
        aria-label="Number of Phases"
        required={chargingEnabled}
        disabled={!chargingEnabled}
        value={numPhases}
        onChange={e => setNumPhases(Number(e.target.value))}>
        <option value="1">uniphase</option>
        <option value="3">three-phase</option>
      </Form.Select>
      <Form.Control plaintext={true} readOnly={true} defaultValue={'Up to ' + getMaxChargingPower()} />
      <Form.Check // prettier-ignore
        type="switch"
        label="Charge on surplus of solar energy"
        checked={chargeOnSurplus}
        onChange={e => setChargeOnSurplus(e.target.checked)}
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
        label="Charge on low Tibber price"
        checked={chargeOnTibber}
        onChange={e => setChargeOnTibber(e.target.checked)}
      />
      <InputGroup className="mb-3">
        <Form.Control
          placeholder="Maximum Tibber Price"
          aria-label="Maximum Tibber Price"
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
      <InputGroup className="mb-3">
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
      <Button type="submit" variant="primary">Save</Button>
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
      <Accordion.Header>Manual Control</Accordion.Header>
      <Accordion.Body>
        <p>
          <Button variant="secondary" onClick={() => manualControl("wakeUp", -1, -1)}>Wake Up</Button>
        </p>
        <p>
          <Button variant="secondary" onClick={() => manualControl("chargeStart", -1, -1)}>Start Charging</Button>
        </p>
        <p>
          <Button variant="secondary" onClick={() => manualControl("chargeStop", -1, -1)}>Stop Charging</Button>
        </p>
        <p>
          Percent:
          <input type="number" value={manCtrlLimit} onChange={e => setManCtrlLimit(Number(e.target.value))} min={1} max={100} />
          <Button variant="secondary" onClick={() => manualControl("chargeLimit", manCtrlLimit, -1)}>Set Charge Limit</Button>
        </p>
        <p>
          Amps:
          <input type="number" value={manCtrlAmps} onChange={e => setManCtrlAmps(Number(e.target.value))} min={0} max={16} />
          <Button variant="secondary" onClick={() => manualControl("chargeAmps", manCtrlAmps, -1)}>Set Charge Amps</Button>
        </p>
        <p>
          <input type="checkbox" checked={manCtrlEnabled === 1} onChange={e => setManCtrlEnabled(e.target.checked ? 1 : 0)} /> Enabled, mins after midnight:
          <input type="number" value={manCtrlMins} onChange={e => setManCtrlMins(Number(e.target.value))} min={0} max={1440} />
          <Button variant="secondary" onClick={() => manualControl("scheduledCharging", manCtrlEnabled, manCtrlMins)}>Set Scheduled Charging</Button>
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
            {token}
            <div>
              {tokenHelp}
              <Button variant="primary" onClick={e => setShowTokenHelp(true)}>How to use</Button>
            </div>
          </Accordion.Body>
        </Accordion.Item>
        {accordionState}
        {accordionSurpluses}
        {accordionChargingEvents}
        {/*accordionManualControl*/}
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
