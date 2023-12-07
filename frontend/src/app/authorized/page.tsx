'use client'

import Link from "next/link";
import { checkAuth, deleteAPI, getAPI, getAccessToken, getBaseUrl, postAPI, putAPI } from "../util";
import { useEffect, useState } from "react";
import Loading from "../loading";
import { Accordion, Button, Form, InputGroup, ListGroup, Modal, Table } from "react-bootstrap";
import { CopyBlock } from "react-code-blocks";

export default function Authorized() {
  const [vehicles, setVehicles] = useState([] as any[])
  const [isLoading, setLoading] = useState(true)
  const [chargingEnabled, setChargingEnabled] = useState(new Map<number, boolean>())
  const [targetSoC, setTargetSoC] = useState(new Map<number, number>())
  const [chargeOnSurplus, setChargeOnSurplus] = useState(new Map<number, boolean>())
  const [minSurplus, setMinSurplus] = useState(new Map<number, number>())
  const [minChargetime, setMinChargetime] = useState(new Map<number, number>())
  const [chargeOnTibber, setChargeOnTibber] = useState(new Map<number, boolean>())
  const [maxPrice, setMaxPrice] = useState(new Map<number, number>())
  const [tibberToken, setTibberToken] = useState(new Map<number, string>())
  const [showTokenHelp, setShowTokenHelp] = useState(new Map<number, boolean>())
  const [surpluses, setSurpluses] = useState(new Map<number, any>())

  function updateChargingEnabled(id: number, value: boolean) {
    let res = new Map(chargingEnabled);
    res.set(id, value);
    setChargingEnabled(res);
  }

  function updateTargetSoC(id: number, value: number) {
    let res = new Map(targetSoC);
    res.set(id, value);
    setTargetSoC(res);
  }

  function updateChargeOnSurplus(id: number, value: boolean) {
    let res = new Map(chargeOnSurplus);
    res.set(id, value);
    setChargeOnSurplus(res);
  }

  function updateMinSurplus(id: number, value: number) {
    let res = new Map(minSurplus);
    res.set(id, value);
    setMinSurplus(res);
  }

  function updateMinChargetime(id: number, value: number) {
    let res = new Map(minChargetime);
    res.set(id, value);
    setMinChargetime(res);
  }

  function updateChargeOnTibber(id: number, value: boolean) {
    let res = new Map(chargeOnTibber);
    res.set(id, value);
    setChargeOnTibber(res);
  }

  function updateTibberToken(id: number, value: string) {
    let res = new Map(tibberToken);
    res.set(id, value);
    setTibberToken(res);
  }

  function updateMaxPrice(id: number, value: number) {
    let res = new Map(maxPrice);
    res.set(id, value);
    setMaxPrice(res);
  }

  function updateShowTokenHelp(id: number, value: boolean) {
    let res = new Map(showTokenHelp);
    res.set(id, value);
    setShowTokenHelp(res);
  }

  const loadLatestSurpluses = async (id: number, token: string) => {
    if (!token) {
      return
    }
    const json = await getAPI("/api/1/user/" + token + "/surplus");
    let res = new Map(surpluses);
    res.set(id, json);
    setSurpluses(res);
  }

  const loadVehicles = async () => {
    const json = await getAPI("/api/1/tesla/my_vehicles");
    setVehicles(json);
    (json as any[]).forEach(e => {
      updateChargingEnabled(e.id, e.enabled);
      updateTargetSoC(e.id, e.target_soc);
      updateChargeOnSurplus(e.id, e.surplus_charging);
      updateMinSurplus(e.id, e.min_surplus);
      updateMinChargetime(e.id, e.min_chargetime);
      updateChargeOnTibber(e.id, e.lowcost_charging);
      updateMaxPrice(e.id, e.max_price);
      updateTibberToken(e.id, e.tibber_token);
      updateShowTokenHelp(e.id, false);
      loadLatestSurpluses(e.id, e.api_token);
    });
  }

  useEffect(() => {
    const fetchData = async () => {
      await checkAuth();
      await loadVehicles();
      setLoading(false);
    }
    fetchData();
  }, []);

  function generateAPIToken(id: string) {
    const fetchData = async () => {
      const json = await postAPI("/api/1/tesla/api_token_create", { vehicle_id: id });
      let vehiclesNew: any[] = [];
      if (vehicles) {
        (vehicles as any[]).forEach(e => {
          if (e.id === id) {
            e.api_token = json.token;
            e.api_password = json.password;
          }
          vehiclesNew.push(e);
        });
      }
      setVehicles(vehiclesNew);
    }
    fetchData();
  }

  function updateAPITokenPassword(id: string) {
    const fetchData = async () => {
      const json = await postAPI("/api/1/tesla/api_token_update/" + id, {});
      let vehiclesNew: any[] = [];
      if (vehicles) {
        (vehicles as any[]).forEach(e => {
          if (e.api_token === id) {
            e.api_token = json.token;
            e.api_password = json.password;
          }
          vehiclesNew.push(e);
        });
      }
      setVehicles(vehiclesNew);
    }
    fetchData();
  }

  function saveVehicle(id: number) {
    const fetchData = async () => {
      setLoading(true);
      let payload = {
        "enabled": chargingEnabled.get(id),
        "target_soc": targetSoC.get(id),
        "surplus_charging": chargeOnSurplus.get(id),
        "min_surplus": minSurplus.get(id),
        "min_chargetime": minChargetime.get(id),
        "lowcost_charging": chargeOnTibber.get(id),
        "max_price": maxPrice.get(id),
        "tibber_token": tibberToken.get(id)
      };
      await putAPI("/api/1/tesla/vehicle_update/" + id, payload);
      await loadVehicles();
      setLoading(false);
    }
    fetchData();
  }

  function deleteVehicle(id: number) {
    if (!window.confirm("Delete his vehicle?")) {
      return;
    }
    const fetchData = async () => {
      setLoading(true);
      await deleteAPI("/api/1/tesla/vehicle_delete/" + id);
      await loadVehicles();
      setLoading(false);
    }
    fetchData();
  }

  if (isLoading) {
    return <Loading />
  }

  let vehicleList = (
    <>
      <p>No vehicles added to your account yet.</p>
      <Link className="btn btn-primary" href="/addvehicle">Add vehicle</Link>
    </>
  );
  if (vehicles && vehicles.length > 0) {
    vehicleList = (
      <ListGroup>
        {(vehicles as any[]).map(v => {

          let code1 = "curl --header 'Content-Type: application/json' --data '{\"password\": \"\", \"surplus_watts\": 1500}' https://tgc.virtualzone.de/api/1/user/" + v.api_token + "/surplus";
          let code2 = "curl --header 'Content-Type: application/json' --data '{\"password\": \"\", \"inverter_active_power_watts\": 2000, \"consumption_watts\": 200}' https://tgc.virtualzone.de/api/1/user/" + v.api_token + "/surplus";
          let code3 = "curl --header 'Content-Type: application/json' --data '{\"password\": \"\"}' https://tgc.virtualzone.de/api/1/user/" + v.api_token + "/plugged_in";
          let code4 = "curl --header 'Content-Type: application/json' --data '{\"password\": \"\"}' https://tgc.virtualzone.de/api/1/user/" + v.api_token + "/unplugged";
          let tokenHelp = (
            <Modal show={showTokenHelp.get(v.id)} onHide={() => updateShowTokenHelp(v.id, false)}>
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
          if (surpluses.get(v.id) && surpluses.get(v.id).length > 0) {
            surplusRows = surpluses.get(v.id).map((s: any) => {
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
                  <th>Time</th>
                  <th>Surplus</th>
                </tr>
              </thead>
              <tbody>
                {surplusRows}
              </tbody>
            </Table>
          );

          let token = <Button variant="link" onClick={() => generateAPIToken(v.id)}>Generate API Token</Button>
          if (v.api_token) {
            token = <>
              API Token: {v.api_token}
              <br />
              Password: ****************
              <Button variant="link" onClick={() => updateAPITokenPassword(v.api_token)}>Update</Button>
            </>
            if (v.api_password) {
              token = <>API Token: {v.api_token}<br /> Password: {v.api_password}</>
            }
          }
          let chargePrefs = (
            <Form onSubmit={e => { e.preventDefault(); e.stopPropagation(); saveVehicle(v.id) }}>
              <Form.Check // prettier-ignore
                type="switch"
                label="Enable smart charging control"
                checked={chargingEnabled.get(v.id)}
                onChange={e => updateChargingEnabled(v.id, e.target.checked)}
              />
              <InputGroup className="mb-3">
                <Form.Control
                  placeholder="Target SoC"
                  aria-label="Target SoC"
                  aria-describedby="target-soc-addon1"
                  type="number"
                  min={1}
                  max={100}
                  required={chargingEnabled.get(v.id)}
                  disabled={!chargingEnabled.get(v.id)}
                  value={targetSoC.get(v.id)}
                  onChange={e => updateTargetSoC(v.id, Number(e.target.value))}
                />
                <InputGroup.Text id="target-soc-addon1">%</InputGroup.Text>
              </InputGroup>
              <Form.Check // prettier-ignore
                type="switch"
                label="Charge on surplus of solar energy"
                checked={chargeOnSurplus.get(v.id)}
                onChange={e => updateChargeOnSurplus(v.id, e.target.checked)}
              />
              <InputGroup className="mb-3">
                <Form.Control
                  placeholder="Minimum surplus"
                  aria-label="Minimum surplus"
                  aria-describedby="min-surplus-addon1"
                  type="number"
                  min={1}
                  max={10000}
                  required={chargeOnSurplus.get(v.id)}
                  disabled={!chargeOnSurplus.get(v.id)}
                  value={minSurplus.get(v.id)}
                  onChange={e => updateMinSurplus(v.id, Number(e.target.value))}
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
                  required={chargeOnSurplus.get(v.id)}
                  disabled={!chargeOnSurplus.get(v.id)}
                  value={minChargetime.get(v.id)}
                  onChange={e => updateMinChargetime(v.id, Number(e.target.value))}
                />
                <InputGroup.Text id="chargetime-addon1">Minutes</InputGroup.Text>
              </InputGroup>
              <Form.Check // prettier-ignore
                type="switch"
                label="Charge on low Tibber price"
                checked={chargeOnTibber.get(v.id)}
                onChange={e => updateChargeOnTibber(v.id, e.target.checked)}
              />
              <InputGroup className="mb-3">
                <Form.Control
                  placeholder="Maximum Tibber Price"
                  aria-label="Maximum Tibber Price"
                  aria-describedby="tibber-price-addon1"
                  type="number"
                  min={1}
                  max={100}
                  required={chargeOnTibber.get(v.id)}
                  disabled={!chargeOnTibber.get(v.id)}
                  value={maxPrice.get(v.id)}
                  onChange={e => updateMaxPrice(v.id, Number(e.target.value))}
                />
                <InputGroup.Text id="tibber-price-addon1">Cents</InputGroup.Text>
              </InputGroup>
              <InputGroup className="mb-3">
                <Form.Control
                  placeholder="Tibber Token"
                  aria-label="Tibber Token"
                  type="text"
                  required={chargeOnTibber.get(v.id)}
                  disabled={!chargeOnTibber.get(v.id)}
                  value={tibberToken.get(v.id)}
                  onChange={e => updateTibberToken(v.id, e.target.value)}
                />
              </InputGroup>
              <Button type="submit" variant="primary">Save</Button>
            </Form>
          );
          return (
            <ListGroup.Item key={v.id}>
              <strong>{v.display_name}</strong>
              <Button variant="danger" size="sm" onClick={e => deleteVehicle(v.id)}>Delete</Button>
              <br />
              {v.vin}
              <Accordion defaultActiveKey="-1">
                <Accordion.Item eventKey="0">
                  <Accordion.Header>API Token</Accordion.Header>
                  <Accordion.Body>
                    {token}
                    <div>
                      {tokenHelp}
                      <Button variant="primary" onClick={e => updateShowTokenHelp(v.id, true)}>How to use</Button>
                    </div>
                  </Accordion.Body>
                </Accordion.Item>
                <Accordion.Item eventKey="1">
                  <Accordion.Header>Charging Preferences</Accordion.Header>
                  <Accordion.Body>
                    {chargePrefs}
                  </Accordion.Body>
                </Accordion.Item>
                <Accordion.Item eventKey="2">
                  <Accordion.Header>Latest recorded surpluses</Accordion.Header>
                  <Accordion.Body>
                    {surplusTable}
                  </Accordion.Body>
                </Accordion.Item>
              </Accordion>
            </ListGroup.Item>
          )
        })}
      </ListGroup>
    );
  }

  return (
    <main>
      <ul>
        <li><a href="https://tesla.com/_ak/tgc.virtualzone.de" target="_blank">Set Up Virtual Key</a></li>
        <li><Link href="/addvehicle">Add vehicle</Link></li>
      </ul>
      <h3>My vehicles</h3>
      {vehicleList}
    </main>
  )
}
