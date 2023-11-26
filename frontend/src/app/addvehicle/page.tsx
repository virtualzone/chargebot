'use client'

import { useEffect, useState } from "react";
import { checkAuth, getAPI, postAPI } from "../util";
import Loading from "../loading";
import NoData from "../nodata";
import Link from "next/link";
import { useRouter } from "next/navigation";

export default function Authorized() {
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

  function selectVehicle(id: string) {
    const fetchData = async () => {
      setLoading(true);
      await postAPI("/api/1/tesla/vehicle_add/" + id, {});
      setLoading(false);
      router.push("/authorized")
    };
    fetchData();
  }

  if (isLoading) {
    return <Loading />
  }

  if (!vehicles) {
    return <NoData />
  }

  return (
    <main className="">
      <div className="pt-6 md:p-8 text-center md:text-left space-y-4">
        <Link href="/authorized">Back</Link>
      </div>
      <div className="pt-6 md:p-8 text-center md:text-left space-y-4">
        <p className="text-lg font-medium">Select the Tesla you want to charge with green power:</p>
      </div>
      <ul className="divide-y divide-slate-100">
        {(vehicles as any[]).map(e => {
          return (
            <article onClick={() => selectVehicle(e.vehicle_id)} key={e.vehicle_id} className="flex items-start space-x-6 p-6">
              <li>{e.display_name} ({e.vin})</li>
            </article>
          )
        })}
      </ul>
    </main>
  )
}
