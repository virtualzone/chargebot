'use client'

import Link from "next/link";
import { checkAuth, deleteAPI, getAPI } from "../util";
import { useEffect, useState } from "react";
import Loading from "../loading";
import { Alert, Container, ListGroup } from "react-bootstrap";
import { useRouter } from "next/navigation";

export default function PageAuthorized() {
  const [vehicles, setVehicles] = useState([] as any[])
  const [isLoading, setLoading] = useState(true)
  const [showAlertAdded, setShowAlertAdded] = useState(false)
  const [showAlertRemoved, setShowAlertRemoved] = useState(false)
  const router = useRouter();

  const loadVehicles = async () => {
    const json = await getAPI("/api/1/tesla/my_vehicles");
    setVehicles(json);
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
      await checkAuth();
      await loadVehicles();
      setLoading(false);
    }
    fetchData();
  }, []);

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

  function selectVehicle(id: number) {
    console.log("/vehicle/?id=" + id);
    router.push("/vehicle/?id=" + id);
  }

  if (vehicles && vehicles.length > 0) {
    vehicleList = (
      <>
        <ListGroup className="mb-5">
          {(vehicles as any[]).map(e => {
            return (
              <ListGroup.Item action={true} onClick={() => selectVehicle(e.id)} key={e.id}>
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

  return (
    <Container fluid="sm" className="pt-5 container-max-width min-height">
      <h2 className="pb-3">My vehicles</h2>
      <Alert variant='success' dismissible={true} hidden={!showAlertAdded}>Vehicle successfully added to your account.</Alert>
      <Alert variant='success' dismissible={true} hidden={!showAlertRemoved}>Vehicle successfully removed from your account.</Alert>
      {vehicleList}
    </Container>
  )
}
