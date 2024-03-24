'use client'

import Link from "next/link";
import { getAPI, postAPI } from "./util";
import { useEffect, useState } from "react";
import Loading from "./loading";
import { Alert, Button, Container, ListGroup, Table } from "react-bootstrap";
import { useRouter } from "next/navigation";

export default function PageAuthorized() {
  const [vehicles, setVehicles] = useState([] as any[])
  const [isLoading, setLoading] = useState(true)
  const [permanentError, setPermanentError] = useState(false)
  const [showAlertAdded, setShowAlertAdded] = useState(false)
  const [showAlertRemoved, setShowAlertRemoved] = useState(false)
  const [surpluses, setSurpluses] = useState([] as any)
  const router = useRouter();

  const loadVehicles = async () => {
    const json = await getAPI("/api/1/tesla/my_vehicles");
    setVehicles(json);
  }

  const loadLatestSurpluses = async () => {
    const json = await getAPI("/api/1/tesla/surplus");
    setSurpluses(json);
  }

  useEffect(() => {
    const searchParams = new URLSearchParams(window.location.search);
    if (searchParams.get("added") === '1') {
      setShowAlertAdded(true);
    }
    if (searchParams.get("removed") === '1') {
      setShowAlertRemoved(true);
    }
    const fetchData = async () => {
      loadVehicles();
      loadLatestSurpluses();
      setPermanentError(await getAPI("/api/1/tesla/permanent_error"))
      setLoading(false);
    }
    fetchData();
  }, []);

  function selectVehicle(vin: string) {
    router.push("/vehicle/?vin=" + vin);
  }

  function resetPermanentError() {
      postAPI("/api/1/tesla/resolve_permanent_error", {});
      setPermanentError(false);
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
      <>
        <ListGroup className="mb-5">
          {(vehicles as any[]).map(e => {
            return (
              <ListGroup.Item action={true} onClick={() => selectVehicle(e.vin)} key={e.vin}>
                <strong>{e.display_name}</strong>
                <br />
                {e.vin}
                <br />
              </ListGroup.Item>
            )
          })}
        </ListGroup>
        <Link className="btn btn-primary" href="/addvehicle">Add vehicle</Link>
      </>
    );
  }

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
  let surplusSection = (
    <>
      <h2 className="pb-3" style={{ 'marginTop': '50px' }}>Latest recorded surpluses</h2>
      {surplusTable}
    </>
  );

  let permanentErrorSection = <></>;
  if (permanentError) {
    permanentErrorSection = (
      <Alert variant='danger' dismissible={false} hidden={!permanentError}>
        <p><strong>Action required</strong></p>
        <p>There was was a recurring error controlling the charging process of one your vehicles. This can happen i.e. if your charger has a problem. chargebot.io has stopped trying to control your vehicle in order to avoid keeping your vehicle awake.</p>
        <p>Try to reconnect your charger and check if everything is set up correctly.</p>
        <p>When done, reset this permanent error in order to start charge-controlling your vehicle again.</p>
        <Button onClick={() => resetPermanentError()}>Reset error</Button>
      </Alert>
    );
  }

  return (
    <Container fluid="sm" className="pt-5 container-max-width min-height">
      <h2 className="pb-3">My vehicles</h2>
      {permanentErrorSection}
      <Alert variant='success' dismissible={true} hidden={!showAlertAdded}>Vehicle successfully added to your account.</Alert>
      <Alert variant='success' dismissible={true} hidden={!showAlertRemoved}>Vehicle successfully removed from your account.</Alert>
      {vehicleList}
      {surplusSection}
    </Container>
  )
}