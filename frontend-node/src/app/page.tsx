'use client'

import Link from "next/link";
import { getAPI, postAPI } from "./util";
import { useEffect, useState } from "react";
import Loading from "./loading";
import { Alert, Button, Container, ListGroup } from "react-bootstrap";
import { useRouter } from "next/navigation";
import VehicleStatus from "./vehicle-status";
import dynamic from "next/dynamic";
const Chart = dynamic(() => import("react-apexcharts"), { ssr: false });

export default function PageAuthorized() {
  const [vehicles, setVehicles] = useState([] as any[])
  const [isLoading, setLoading] = useState(true)
  const [permanentError, setPermanentError] = useState(false)
  const [showAlertAdded, setShowAlertAdded] = useState(false)
  const [showAlertRemoved, setShowAlertRemoved] = useState(false)
  const [surplusChartOptions, setSurplusChartOptions] = useState({} as ApexCharts.ApexOptions)
  const [surplusChartSeries, setSurplusChartSeries] = useState([] as ApexCharts.ApexOptions["series"])
  const router = useRouter();

  const loadVehicles = async () => {
    const json = await getAPI("/api/1/tesla/my_vehicles");
    setVehicles(json);
  }

  const loadLatestSurpluses = async () => {
    const json = await getAPI("/api/1/tesla/surplus");
    const htmlElement = document.querySelector("html");
    let darkMode = false;
    if (htmlElement !== null) {
      darkMode = (htmlElement.getAttribute("data-bs-theme") === 'dark');
    }
    if ((json === 'undefined') || (json.length === 0)) {
      return;
    }
    let options = {
      chart: {
        toolbar: {
          show: false
        },
        id: "surplus-chart",
        background: darkMode ? 'rgb(33, 37, 41)' : undefined,
        zoom: {
          enabled: false
        }
      },
      stroke: {
        width: 1
      },
      xaxis: {
        categories: json.map((s: any) => s.ts.replace('T', ' ').replace('Z', '')).reverse(),
      },
      theme: {
        mode: darkMode ? 'dark' as 'dark': 'light' as 'light'
      }
    };
    let series = [
      {
        name: "Surplus (Watts)",
        data: json.map((s: any) => s.surplus_watts).reverse()
        
      }
    ];
    setSurplusChartOptions(options);
    setSurplusChartSeries(series);
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
              <ListGroup.Item action={true} onClick={() => selectVehicle(e.vehicle.vin)} key={e.vehicle.vin}>
                <VehicleStatus vehicle={e.vehicle} state={e.state} />
              </ListGroup.Item>
            )
          })}
        </ListGroup>
      </>
    );
  }

  let surplusTable = (
    <Chart type="line" width="100%" height={300} options={surplusChartOptions} series={surplusChartSeries} />
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
      <h2 className="pb-3">
        My vehicles
        <Link className="btn btn-secondary btn-sm" href="/addvehicle" style={{'float': 'right'}}>+</Link>
      </h2>
      {permanentErrorSection}
      <Alert variant='success' dismissible={true} hidden={!showAlertAdded}>Vehicle successfully added to your account.</Alert>
      <Alert variant='success' dismissible={true} hidden={!showAlertRemoved}>Vehicle successfully removed from your account.</Alert>
      {vehicleList}
      {surplusSection}
    </Container>
  )
}