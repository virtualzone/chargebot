'use client'

import Image from "next/image";
import { checkAuth, getAPI, getUserDetails, postAPI, saveUserDetails } from "../util";
import { useEffect, useState } from "react";
import Loading from "../loading";
import { Alert, Button, Container, Form, InputGroup } from "react-bootstrap";
import { Loader as IconLoad, Navigation as IconLocation } from 'react-feather';
import { copyToClipboard } from "../util";

export default function PageAuthorized() {
  const [vehicles, setVehicles] = useState([] as any[])
  const [isLoading, setLoading] = useState(true)
  const [showAlertAdded, setShowAlertAdded] = useState(false)
  const [showAlertRemoved, setShowAlertRemoved] = useState(false)
  const [teslaAccountLinked, setTeslaAccountLinked] = useState(false)
  const [savingApiToken, setSavingApiToken] = useState(false)
  const [apiToken, setApiToken] = useState('')
  const [apiTokenPassword, setApiTokenPassword] = useState('')
  const [homeLatitude, setHomeLatitude] = useState(0.0)
  const [homeLongitude, setHomeLongitude] = useState(0.0)
  const [homeRadius, setHomeRadius] = useState(100)
  const [savingHomeLocation, setSavingHomeLocation] = useState(false)

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
      let userDetails = await getAPI("/api/1/auth/me");
      saveUserDetails(userDetails);
      setApiToken(userDetails.api_token);
      if (userDetails.tesla_user_id) {
        setTeslaAccountLinked(true);
        await loadVehicles();
      } else {
        setTeslaAccountLinked(false);
      }
      setHomeLatitude(userDetails.home_lat);
      setHomeLongitude(userDetails.home_lng);
      setHomeRadius(userDetails.home_radius);
      setLoading(false);
    }
    fetchData();
  }, []);

  async function linkTeslaAccount() {
    const json = await getAPI("/api/1/auth/tesla/init3rdparty");
    if (typeof window !== "undefined") {
      window.location.href = json.url;
    }
  }

  async function saveHomeLocation() {
    const fetchData = async () => {
      setSavingHomeLocation(true);
      let payload = {
        "lat": homeLatitude,
        "lng": homeLongitude,
        "radius": homeRadius
      };
      await postAPI("/api/1/auth/home", payload);
      let user = await getAPI("/api/1/auth/me");
      saveUserDetails(user);
      setSavingHomeLocation(false);
    }
    fetchData();
  }

  function getGeoLocation() {
    const cb = function (position: GeolocationPosition): void {
      setHomeLatitude(position.coords.latitude);
      setHomeLongitude(position.coords.longitude);
    }
    if ((navigator) && (navigator.geolocation)) {
      navigator.geolocation.getCurrentPosition(cb);
    }
  }

  function updateAPITokenPassword(id: string) {
    const fetchData = async () => {
      setSavingApiToken(true);
      const json = await postAPI("/api/1/tesla/api_token_update/" + id, {});
      setApiToken(json.token);
      setApiTokenPassword(json.password);
      setSavingApiToken(false);
    }
    if (confirm('Do you really want to replace the existing password?')) {
      fetchData();
    }
  }

  if (isLoading) {
    return <Loading />
  }

  let vehicleList = (
    <>
      <p>Your Tesla account is linked with your chargebot.io account.</p>
      <p>Set up your remote controller node in order to control your vehicle's charging process.</p>
      <p>When necessary, you can re-link your Tesla account and get a Token again.</p>
      <Button variant="danger" onClick={() => linkTeslaAccount()}>
        <Image src="/tesla-icon.svg" width={24} height={24} alt="" className="me-2" />
        Re-Link your Tesla Account
      </Button>
    </>
  );

  if (!teslaAccountLinked) {
    vehicleList = (
      <>
        <p>Welcome to chargebot.io!</p>
        <p>First, we'll need to link your Tesla account with your chargebot.io account.</p>
        <p>This will generate a Token which is required on your remote controller node in order to control your vehicle's charging process. The Token is neither saved nor used directly by chargebot.io.</p>
        <Button variant="danger" onClick={() => linkTeslaAccount()}>
          <Image src="/tesla-icon.svg" width={24} height={24} alt="" className="me-2" />
          Link your Tesla Account
        </Button>
      </>
    );
  }

  let token = <>
    <strong>API Token:</strong>
    <Button variant="link" onClick={() => copyToClipboard(apiToken)}>Copy</Button>
    <br />
    <pre>{apiToken}</pre>
    <strong>Password:</strong>
    <Button variant="link" onClick={() => updateAPITokenPassword(apiToken)} disabled={savingApiToken}>{savingApiToken ? <><IconLoad className="feather-button loader" /> Updating...</> : 'Update'}</Button>
    <br />
    <pre>****************</pre>
  </>
  if (apiTokenPassword) {
    token = <>
      <strong>API Token:</strong>
      <Button variant="link" onClick={() => copyToClipboard(apiToken)}>Copy</Button>
      <br />
      <pre>{apiToken}</pre>
      <strong>Password:</strong>
      <Button variant="link" onClick={() => copyToClipboard(apiTokenPassword)}>Copy</Button>
      <br />
      <pre>{apiTokenPassword}</pre>
    </>
  }
  let tokenSection = (
    <>
      <h2 className="pb-3" style={{ 'marginTop': '50px' }}>API Token</h2>
      <p>An individual API token is required to connect your remote controller node to chargebot.io.</p>
      {token}
    </>
  );

  let vehiclesSection = <></>;
  if (vehicles) {
    if (vehicles.length === 0) {
      vehiclesSection = (
        <>
          <h2 className="pb-3" style={{ 'marginTop': '50px' }}>Vehicles</h2>
          <p>No vehicles have been assigned to your account yet. When setting up your remote controller node and adding vehicles, these get assigned to your account automatically.</p>
        </>
      );
    } else {
      vehiclesSection = (
        <>
          <h2 className="pb-3" style={{ 'marginTop': '50px' }}>Vehicles</h2>
          <p>The following vehicles have been assigned to your account automatically using your remote controller node:</p>
          <ul>
            {vehicles.map(v => {
              return <li key={v.vin}>{v.vin}</li>
            })}
          </ul>
        </>
      );
    }
  }

  let homeLocation = (
    <>
      <h2 className="pb-3" style={{ 'marginTop': '50px' }}>Home Location</h2>
      <p>Your home location is required so that chargebot.io recognizes whether your vehicle is plugged in at home.</p>
      <Form onSubmit={e => { e.preventDefault(); e.stopPropagation(); saveHomeLocation() }}>
        <InputGroup className="mb-3">
          <Form.Control
            placeholder="Latitude"
            aria-label="Latitude"
            type="text"
            min={0.0}
            max={100.0}
            required={true}
            value={homeLatitude}
            onChange={e => setHomeLatitude(Number(e.target.value))}
          />
        </InputGroup>
        <InputGroup className="mb-3">
          <Form.Control
            placeholder="Longitude"
            aria-label="Longitude"
            type="text"
            min={0.0}
            max={100.0}
            required={true}
            value={homeLongitude}
            onChange={e => setHomeLongitude(Number(e.target.value))}
          />
        </InputGroup>
        <InputGroup className="mb-3">
          <Form.Control
            placeholder="Longitude"
            aria-label="Longitude"
            aria-describedby="lng-addon1"
            type="number"
            min={5}
            max={100}
            step={1}
            required={true}
            value={homeRadius}
            onChange={e => setHomeRadius(Number(e.target.value))}
          />
          <InputGroup.Text id="lng-addon1">m</InputGroup.Text>
        </InputGroup>
        <Button type="button" variant="secondary" style={{ 'marginRight': '5px' }} onClick={() => { getGeoLocation(); }}><IconLocation className="feather-button" /></Button>
        <Button type="submit" variant="primary" disabled={savingHomeLocation}>{savingHomeLocation ? <><IconLoad className="feather-button loader" /> Saving...</> : 'Save'}</Button>
      </Form>
    </>
  );

  return (
    <Container fluid="sm" className="pt-5 container-max-width min-height">
      <h2 className="pb-3">My account</h2>
      <Alert variant='success' dismissible={true} hidden={!showAlertAdded}>Vehicle successfully added to your account.</Alert>
      <Alert variant='success' dismissible={true} hidden={!showAlertRemoved}>Vehicle successfully removed from your account.</Alert>
      {vehicleList}
      {tokenSection}
      {homeLocation}
      {vehiclesSection}
    </Container>
  )
}
