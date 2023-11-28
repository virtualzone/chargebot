'use client'

import Link from "next/link";
import { checkAuth, getAPI, getAccessToken, getBaseUrl, postAPI } from "../util";
import { useEffect, useState } from "react";
import Loading from "../loading";
import { Button, ListGroup } from "react-bootstrap";

export default function Authorized() {
  const [vehicles, setVehicles] = useState([] as any[])
  const [isLoading, setLoading] = useState(true)

  useEffect(() => {
    const fetchData = async () => {
      await checkAuth();
      const json = await getAPI("/api/1/tesla/my_vehicles");
      setVehicles(json);
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
      const json = await postAPI("/api/1/tesla/api_token_update/"+id, {});
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

  if (isLoading) {
    return <Loading />
  }

  let vehicleList = <>No vehicles added to your account yet.</>
  if (vehicles) {
    vehicleList = (
      <ListGroup>
        {(vehicles as any[]).map(e => {
          let token = <Button variant="link" onClick={() => generateAPIToken(e.id)}>Generate API Token</Button>
          if (e.api_token) {
            token = <>
              API Token: {e.api_token}
              <br />
              Password: ****************
              <Button variant="link" onClick={() => updateAPITokenPassword(e.api_token)}>Update</Button>
            </>
            if (e.api_password) {
              token = <>API Token: {e.api_token}<br /> Password: {e.api_password}</>
            }
          }
          return (
            <ListGroup.Item key={e.id}>
              <strong>{e.display_name}</strong><br />
              {e.vin}<br />
              {token}
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
