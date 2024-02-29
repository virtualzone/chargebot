'use client'

import { useEffect, useState } from "react";
import { checkAuth, getAPI, postAPI } from "../util";
import Loading from "../loading";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Container, ListGroup } from "react-bootstrap";

export default function PageAddVehicle() {
  const [vehicles, setVehicles] = useState([] as any[])
  const [isLoading, setLoading] = useState(true)
  const router = useRouter();

  useEffect(() => {
    const fetchData = async () => {
      await checkAuth();
      const vehicles = await getAPI("/api/1/tesla/vehicles");
      const myVehicles = await getAPI("/api/1/tesla/my_vehicles");
      const finalList: any[] = [];
      (vehicles as any[]).forEach(e => {
        let found = false;
        (myVehicles as any[]).forEach(m => {
          if (m.id === e.vehicle_id) {
            found = true;
          }
        });
        if (!found) {
          finalList.push(e);
        }
      });
      setVehicles(finalList);
      setLoading(false);
    }
    fetchData();
  }, []);

  function selectVehicle(vin: string) {
    const fetchData = async () => {
      setLoading(true);
      await postAPI("/api/1/tesla/vehicle_add/" + vin, {});
      setLoading(false);
      router.push("/authorized/?added=1")
    };
    fetchData();
  }

  if (isLoading) {
    return <Loading />
  }

  if ((!vehicles) || (vehicles.length === 0)) {
    return (
      <Container fluid="sm" className="pt-5 container-max-width min-height">
        <h2 className="pb-3">Add vehicle</h2>
        <p>No vehicles found in your Tesla account which can be added to chargebot.io.</p>
        <Link href="/authorized" className="btn btn-link">&lt; Back</Link>
      </Container>
    );
  }

  return (
    <Container fluid="sm" className="pt-5 container-max-width min-height">
      <h2 className="pb-3">Add vehicle</h2>
      <p>Select the Tesla you want to manage with chargebot.io:</p>
      <ListGroup className="mb-5">
        {(vehicles as any[]).map(e => {
          return (
            <ListGroup.Item action={true} onClick={() => selectVehicle(e.vin)} key={e.vin}>
              <strong>{e.display_name}</strong>
              <br />
              {e.vin}
            </ListGroup.Item>
          )
        })}
      </ListGroup>
      <Link href="/authorized" className="btn btn-link">&lt; Back</Link>
    </Container>
  )
}
