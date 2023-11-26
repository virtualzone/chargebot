'use client'

import Link from "next/link";
import { checkAuth, getAPI, getAccessToken, getBaseUrl } from "../util";
import { useEffect, useState } from "react";
import Loading from "../loading";

export default function Authorized() {
  const [vehicles, setVehicles] = useState(null)
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

  if (isLoading) {
    return <Loading />
  }

  let vehicleList = <>No vehicles added to your account yet.</>
  if (vehicles) {
    vehicleList = (
      <>
        {(vehicles as any[]).map(e => {
          return (
            <li key={e.id}>{e.display_name} ({e.vin})</li>
          )
        })}
      </>
    );
  }

  return (
    <main>
      <ul>
        <li><a href="https://tesla.com/_ak/tgc.virtualzone.de" target="_blank">Set Up Virtual Key</a></li>
        <li><Link href="/addvehicle">Add vehicle</Link></li>
      </ul>
      <div>
        My vehicles:
      </div>
      <ul>
        {vehicleList}
      </ul>
    </main>
  )
}
