'use client'

import { deleteAPI, getAPI, postAPI, putAPI } from "../util";
import { useEffect, useState } from "react";
import Loading from "../loading";
import { Accordion, Button, Col, Container, Form, InputGroup, Row, Table } from "react-bootstrap";
import { useRouter } from "next/navigation";
import { Loader as IconLoad } from 'react-feather';
import Link from "next/link";
import VehicleStatus from "../vehicle-status";

export default function PageVehicle() {
  let vehicleVIN = ""
  const router = useRouter()
  const [vehicle, setVehicle] = useState({} as any)
  const [isLoading, setLoading] = useState(true)
  const [savingVehicle, setSavingVehicle] = useState(false)
  const [chargingEnabled, setChargingEnabled] = useState(false)
  const [targetSoC, setTargetSoC] = useState(0)
  const [maxAmps, setMaxAmps] = useState(0)
  const [numPhases, setNumPhases] = useState(0)
  const [chargeOnSurplus, setChargeOnSurplus] = useState(false)
  const [minSurplus, setMinSurplus] = useState(0)
  const [surplusBuffer, setSurplusBuffer] = useState(0)
  const [minChargetime, setMinChargetime] = useState(0)
  const [chargeOnTibber, setChargeOnTibber] = useState(false)
  const [gridProvider, setGridProvider] = useState('tibber')
  const [gridStrategy, setGridStrategy] = useState(1)
  const [maxPrice, setMaxPrice] = useState(0)
  const [departDays, setDepartDays] = useState([1, 2, 3, 4, 5])
  const [departTime, setDepartTime] = useState('07:00')
  const [tibberToken, setTibberToken] = useState('')
  const [vehicleState, setVehicleState] = useState({} as any)
  const [chargingEvents, setChargingEvents] = useState([] as any)
  const [maxChargingPower, setMaxChargingPower] = useState('0 kW')

  useEffect(() => {
    const searchParams = new URLSearchParams(window.location.search);
    const vin = searchParams.get("vin");
    if (!vin) {
      router.push('/')
      return;
    }
    vehicleVIN = vin;
    const fetchData = async () => {
      await loadVehicle(vehicleVIN);
      setLoading(false);
    }
    fetchData();
  }, [router]);

  /*
  const loadVehicleState = async (vin: string) => {
    const json = await getAPI("/api/1/tesla/state/" + vin);
    if (json) {
      setVehicleState(json);
    }
  }
  */

  const loadLatestChargingEvents = async (vin: string) => {
    const json = await getAPI("/api/1/tesla/events/" + vin);
    setChargingEvents(json);
  }

  const setVehicleDetails = (e: any) => {
    setChargingEnabled(e.enabled);
    setTargetSoC(e.target_soc);
    setMaxAmps(e.max_amps);
    setNumPhases(e.num_phases);
    setChargeOnSurplus(e.surplus_charging);
    setMinSurplus(e.min_surplus);
    setSurplusBuffer(e.surplus_buffer);
    setMinChargetime(e.min_chargetime);
    setChargeOnTibber(e.lowcost_charging);
    setMaxPrice(e.max_price);
    setGridProvider(e.gridProvider);
    setGridStrategy(e.gridStrategy);
    setDepartDays([...e.departDays].map(i => Number(i)));
    setDepartTime(e.departTime);
    setTibberToken(e.tibber_token);
  }

  const loadVehicle = async (vin: string) => {
    const e = await getAPI("/api/1/tesla/my_vehicle/" + vin);
    setVehicleDetails(e.vehicle);
    setVehicleState(e.state);
    updateMaxChargingPower(e.vehicle.max_amps, e.vehicle.num_phases);
    //loadVehicleState(e.vin);
    loadLatestChargingEvents(e.vehicle.vin);
    setVehicle(e.vehicle);
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
        "surplus_buffer": surplusBuffer,
        "min_chargetime": minChargetime,
        "lowcost_charging": chargeOnTibber,
        "gridProvider": gridProvider,
        "gridStrategy": gridStrategy,
        "departDays": departDays.join(''),
        "departTime": departTime,
        "max_price": maxPrice,
        "tibber_token": tibberToken
      };
      await putAPI("/api/1/tesla/vehicle_update/" + vehicle.vin, payload);
      await loadVehicle(vehicle.vin);
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
      await deleteAPI("/api/1/tesla/vehicle_delete/" + vehicle.vin);
      router.push('/?removed=1');
    }
    fetchData();
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

  function updateMaxChargingPower(maxAmps: number, numPhases: number) {
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
      setMaxChargingPower(Math.round(p / 1000) + " kW");
      return;
    }
    setMaxChargingPower(p + " W");
  }

  function manualControlTestDrive() {
    postAPI("/api/1/ctrl/" + vehicle.vin + "/testDrive", null).then(res => {
      window.alert('Charging should start shortly. It will be stopped after 30 seconds. Please check your Tesla App if this automation works.');
    })
  }

  if (isLoading) {
    return <Loading />
  }

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

  let chargePrefs = (
    <Form onSubmit={e => { e.preventDefault(); e.stopPropagation(); saveVehicle() }}>
      <Form.Group as={Row}>
        <Col>
          <Form.Check // prettier-ignore
            type="switch"
            label="Enable smart charging control"
            checked={chargingEnabled}
            onChange={e => setChargingEnabled(e.target.checked)}
          />
        </Col>
      </Form.Group>
      <Form.Group as={Row}>
        <Form.Label column={true} className="sm-4">Target SoC: {targetSoC} %</Form.Label>
        <Col sm={8} style={{ 'paddingTop': '7px', 'paddingBottom': '7px' }}><Form.Range min={1} max={100} value={targetSoC} onChange={e => setTargetSoC(Number(e.target.value))} /></Col>
      </Form.Group>
      <Form.Group as={Row}>
        <Form.Label column={true} className="sm-4">Max. Amps: {maxAmps} A</Form.Label>
        <Col sm={8} style={{ 'paddingTop': '7px', 'paddingBottom': '7px' }}><Form.Range min={1} max={32} value={maxAmps} onChange={e => { setMaxAmps(Number(e.target.value)); updateMaxChargingPower(Number(e.target.value), numPhases) }} /></Col>
      </Form.Group>
      <Form.Group as={Row}>
        <Form.Label column={true} className="sm-4">Num. Phases:</Form.Label>
        <Col sm={8}>
          <Form.Select
            aria-label="Number of Phases"
            required={chargingEnabled}
            value={numPhases}
            onChange={e => { setNumPhases(Number(e.target.value)); updateMaxChargingPower(maxAmps, Number(e.target.value)) }}>
            <option value="1">uniphase</option>
            <option value="3">three-phase</option>
          </Form.Select>
        </Col>
      </Form.Group>
      <Form.Group as={Row}>
        <Form.Label column={true} className="sm-4"></Form.Label>
        <Col sm={8}>
          <Form.Control plaintext={true} readOnly={true} value={'Up to ' + maxChargingPower} />
        </Col>
      </Form.Group>
      <Form.Group as={Row}>
        <Col>
          <Form.Check // prettier-ignore
            type="switch"
            label="Charge on surplus of solar energy"
            checked={chargeOnSurplus}
            onChange={e => setChargeOnSurplus(e.target.checked)}
            style={{ 'marginTop': '25px' }}
          />
        </Col>
      </Form.Group>
      <Form.Group as={Row}>
        <Form.Label column={true} className="sm-4">Min. surplus: {minSurplus} W</Form.Label>
        <Col sm={8} style={{ 'paddingTop': '7px', 'paddingBottom': '7px' }}><Form.Range required={chargeOnSurplus} disabled={!chargeOnSurplus} min={230} max={5000} value={minSurplus} onChange={e => setMinSurplus(Number(e.target.value))} /></Col>
      </Form.Group>
      <Form.Group as={Row}>
        <Form.Label column={true} className="sm-4">Buffer: {surplusBuffer} W</Form.Label>
        <Col sm={8} style={{ 'paddingTop': '7px', 'paddingBottom': '7px' }}><Form.Range required={chargeOnSurplus} disabled={!chargeOnSurplus} min={0} max={5000} value={surplusBuffer} onChange={e => setSurplusBuffer(Number(e.target.value))} /></Col>
      </Form.Group>
      <Form.Group as={Row}>
        <Form.Label column={true} className="sm-4">Min. charge time: {minChargetime} m</Form.Label>
        <Col sm={8} style={{ 'paddingTop': '7px', 'paddingBottom': '7px' }}><Form.Range required={chargeOnSurplus} disabled={!chargeOnSurplus} min={0} max={60} value={minChargetime} onChange={e => setMinChargetime(Number(e.target.value))} /></Col>
      </Form.Group>
      <Form.Group as={Row}>
        <Col>
          <Form.Check // prettier-ignore
            type="switch"
            label="Charge on low grid price"
            checked={chargeOnTibber}
            onChange={e => setChargeOnTibber(e.target.checked)}
            style={{ 'marginTop': '25px' }}
          />
        </Col>
      </Form.Group>
      <Form.Group as={Row}>
        <Form.Label column={true} className="sm-4">Grid provider:</Form.Label>
        <Col sm={8}>
          <Form.Select
            aria-label="Provider"
            required={chargeOnTibber}
            disabled={!chargeOnTibber}
            value={gridProvider}
            onChange={e => setGridProvider(e.target.value)}>
            <option value="tibber">Tibber</option>
          </Form.Select>
        </Col>
      </Form.Group>
      <Form.Group as={Row}>
        <Form.Label column={true} className="sm-4">Strategy:</Form.Label>
        <Col sm={8}>
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
        </Col>
      </Form.Group>
      <Form.Group as={Row} hidden={gridStrategy === 3}>
        <Form.Label column={true} className="sm-4">Max. price: {maxPrice} Cents</Form.Label>
        <Col sm={8} style={{ 'paddingTop': '7px', 'paddingBottom': '7px' }}><Form.Range required={chargeOnTibber} disabled={!chargeOnTibber} min={1} max={100} value={maxPrice} onChange={e => setMaxPrice(Number(e.target.value))} /></Col>
      </Form.Group>
      <Form.Group as={Row} hidden={gridStrategy === 1}>
        <Form.Label column={true} className="sm-4"></Form.Label>
        <Col sm={8}>
          <Form.Check inline={true} label="Mo" type="checkbox" disabled={!chargeOnTibber} checked={departDays.indexOf(1) > -1} onChange={e => setDepartDays(e.target.checked ? [...departDays, 1] : [...departDays].toSpliced(departDays.indexOf(1), 1))} />
          <Form.Check inline={true} label="Tu" type="checkbox" disabled={!chargeOnTibber} checked={departDays.indexOf(2) > -1} onChange={e => setDepartDays(e.target.checked ? [...departDays, 2] : [...departDays].toSpliced(departDays.indexOf(2), 1))} />
          <Form.Check inline={true} label="We" type="checkbox" disabled={!chargeOnTibber} checked={departDays.indexOf(3) > -1} onChange={e => setDepartDays(e.target.checked ? [...departDays, 3] : [...departDays].toSpliced(departDays.indexOf(3), 1))} />
          <Form.Check inline={true} label="Th" type="checkbox" disabled={!chargeOnTibber} checked={departDays.indexOf(4) > -1} onChange={e => setDepartDays(e.target.checked ? [...departDays, 4] : [...departDays].toSpliced(departDays.indexOf(4), 1))} />
          <Form.Check inline={true} label="Fr" type="checkbox" disabled={!chargeOnTibber} checked={departDays.indexOf(5) > -1} onChange={e => setDepartDays(e.target.checked ? [...departDays, 5] : [...departDays].toSpliced(departDays.indexOf(5), 1))} />
          <Form.Check inline={true} label="Sa" type="checkbox" disabled={!chargeOnTibber} checked={departDays.indexOf(6) > -1} onChange={e => setDepartDays(e.target.checked ? [...departDays, 6] : [...departDays].toSpliced(departDays.indexOf(6), 1))} />
          <Form.Check inline={true} label="Su" type="checkbox" disabled={!chargeOnTibber} checked={departDays.indexOf(7) > -1} onChange={e => setDepartDays(e.target.checked ? [...departDays, 7] : [...departDays].toSpliced(departDays.indexOf(7), 1))} />
        </Col>
      </Form.Group>
      <Form.Group as={Row} hidden={gridStrategy === 1}>
        <Form.Label column={true} className="sm-4"></Form.Label>
        <Col sm={8}>
          <Form.Control
            type="time"
            min="00:00"
            max="23:59"
            required={chargeOnTibber}
            disabled={!chargeOnTibber}
            value={departTime}
            onChange={e => setDepartTime(e.target.value)}
          />
        </Col>
      </Form.Group>
      <Form.Group as={Row} hidden={gridProvider !== 'tibber'}>
        <Form.Label column={true} className="sm-4">Tibber Token</Form.Label>
        <Col sm={8}>
          <Form.Control
            placeholder="Tibber Token"
            aria-label="Tibber Token"
            type="text"
            required={chargeOnTibber}
            disabled={!chargeOnTibber}
            value={tibberToken}
            onChange={e => setTibberToken(e.target.value)}
          />
          <p><a href="https://developer.tibber.com/settings/accesstoken" target="_blank">Get your Tibber Access Token</a></p>
        </Col>
      </Form.Group>
      <Button type="submit" variant="primary" disabled={savingVehicle}>{savingVehicle ? <><IconLoad className="feather-button loader" /> Saving...</> : 'Save'}</Button>
    </Form>
  );
  let accordionChargingEvents = <></>;
  accordionChargingEvents = (
    <Accordion.Item eventKey="4">
      <Accordion.Header>Latest charging events</Accordion.Header>
      <Accordion.Body>
        {eventsTable}
      </Accordion.Body>
    </Accordion.Item>
  );
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
      <VehicleStatus state={vehicleState} vehicle={vehicle} />
      <Accordion defaultActiveKey={'0'} flush={true} style={{ 'marginTop': '50px' }}>
        <Accordion.Item eventKey="0">
          <Accordion.Header>Charging Preferences</Accordion.Header>
          <Accordion.Body>
            {chargePrefs}
          </Accordion.Body>
        </Accordion.Item>
        <Accordion.Item eventKey="1">
          <Accordion.Header>Setup</Accordion.Header>
          <Accordion.Body>
            <p>Before chargebot.io can control your vehicle's charging process, you need to set up the virtual key:</p>
            <p><a href="https://tesla.com/_ak/chargebot.io" target="_blank">Set Up Virtual Key</a></p>
          </Accordion.Body>
        </Accordion.Item>
        {accordionChargingEvents}
        {accordionManualControl}
        <Accordion.Item eventKey="99">
          <Accordion.Header>Danger zone</Accordion.Header>
          <Accordion.Body>
            <Button variant="danger" onClick={() => deleteVehicle()}>Remove vehicle from chargebot.io</Button>
          </Accordion.Body>
        </Accordion.Item>
      </Accordion>
      <Link href="/" className="btn btn-link">&lt; Back</Link>
    </Container>
  );
}
