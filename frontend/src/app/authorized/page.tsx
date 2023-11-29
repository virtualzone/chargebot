'use client'

import Link from "next/link";
import { checkAuth, deleteAPI, getAPI, getAccessToken, getBaseUrl, postAPI, putAPI } from "../util";
import { useEffect, useState } from "react";
import Loading from "../loading";
import { Accordion, Button, Form, InputGroup, ListGroup } from "react-bootstrap";

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

  function updateMaxPrice(id: number, value: number) {
    let res = new Map(maxPrice);
    res.set(id, value);
    setMaxPrice(res);
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
        "max_price": maxPrice.get(id)
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
            <Form onSubmit={e => { e.preventDefault(); saveVehicle(v.id) }}>
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
                  aria-describedby="basic-addon1"
                  type="number"
                  min={1}
                  max={100}
                  disabled={!chargingEnabled.get(v.id)}
                  value={targetSoC.get(v.id)}
                  onChange={e => updateTargetSoC(v.id, Number(e.target.value))}
                />
                <InputGroup.Text id="basic-addon1">%</InputGroup.Text>
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
                  aria-describedby="basic-addon1"
                  type="number"
                  min={1}
                  max={10000}
                  disabled={!chargeOnSurplus.get(v.id)}
                  value={minSurplus.get(v.id)}
                  onChange={e => updateMinSurplus(v.id, Number(e.target.value))}
                />
                <InputGroup.Text id="basic-addon1">Watts</InputGroup.Text>
              </InputGroup>
              <InputGroup className="mb-3">
                <Form.Control
                  placeholder="Minimum Charging Time"
                  aria-label="Minimum Charging Time"
                  aria-describedby="basic-addon1"
                  type="number"
                  min={1}
                  max={120}
                  disabled={!chargeOnSurplus.get(v.id)}
                  value={minChargetime.get(v.id)}
                  onChange={e => updateMinChargetime(v.id, Number(e.target.value))}
                />
                <InputGroup.Text id="basic-addon1">Minutes</InputGroup.Text>
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
                  aria-describedby="basic-addon1"
                  type="number"
                  min={1}
                  max={100}
                  disabled={!chargeOnTibber.get(v.id)}
                  value={maxPrice.get(v.id)}
                  onChange={e => updateMaxPrice(v.id, Number(e.target.value))}
                />
                <InputGroup.Text id="basic-addon1">Cents</InputGroup.Text>
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
                  </Accordion.Body>
                </Accordion.Item>
                <Accordion.Item eventKey="1">
                  <Accordion.Header>Charging Preferences</Accordion.Header>
                  <Accordion.Body>
                    {chargePrefs}
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
