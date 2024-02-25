'use client'

import Link from "next/link";
import Image from "next/image";
import { checkAuth, getAPI, getUserDetails, postAPI } from "../util";
import { useEffect, useState } from "react";
import Loading from "../loading";
import { Alert, Button, Container, ListGroup, Modal } from "react-bootstrap";
import { useRouter } from "next/navigation";
import { CopyBlock } from "react-code-blocks";
import { Loader as IconLoad } from 'react-feather';

export default function PageAuthorized() {
  const [vehicles, setVehicles] = useState([] as any[])
  const [isLoading, setLoading] = useState(true)
  const [showAlertAdded, setShowAlertAdded] = useState(false)
  const [showAlertRemoved, setShowAlertRemoved] = useState(false)
  const [teslaAccountLinked, setTeslaAccountLinked] = useState(false)
  const [savingApiToken, setSavingApiToken] = useState(false)
  const [showTokenHelp, setShowTokenHelp] = useState(false)
  const [apiToken, setApiToken] = useState('')
  const [apiTokenPassword, setApiTokenPassword] = useState('')
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
      let userDetails = getUserDetails();
      setApiToken(userDetails.api_token);
      if (userDetails.tesla_user_id) {
        setTeslaAccountLinked(true);
        await loadVehicles();
      } else {
        setTeslaAccountLinked(false);
      }
      setLoading(false);
    }
    fetchData();
  }, []);

  function selectVehicle(id: number) {
    console.log("/vehicle/?id=" + id);
    router.push("/vehicle/?id=" + id);
  }

  async function linkTeslaAccount() {
    const json = await getAPI("/api/1/auth/tesla/init3rdparty");
    if (typeof window !== "undefined") {
      window.location.href = json.url;
    }
  }

  async function copyToClipboard(s: string) {
    try {
      await navigator.clipboard.writeText(s);
    } catch (err) {
      console.error('Failed to copy: ', err);
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

  if (!teslaAccountLinked) {
    vehicleList = (
      <>
        <p>Welcome to chargebot.io!</p>
        <p>First, we'll need to link your Tesla account with your chargebot.io account.</p>
        <p>This will generate an Access Token which is required in order to control your vehicle's charging process.</p>
        <Button variant="danger" onClick={() => linkTeslaAccount()}>
          <Image src="/tesla-icon.svg" width={24} height={24} alt="" className="me-2" />
          Link your Tesla Account
        </Button>
      </>
    );
  }

  if (teslaAccountLinked && vehicles && vehicles.length > 0) {
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


  let code1 = "curl --header 'Content-Type: application/json' --data '{\"password\": \"\", \"surplus_watts\": 1500}' https://chargebot.io/api/1/user/" + apiToken + "/{vehicleID}/surplus";
  let code2 = "curl --header 'Content-Type: application/json' --data '{\"password\": \"\", \"inverter_active_power_watts\": 2000, \"consumption_watts\": 200}' https://chargebot.io/api/1/user/" + apiToken + "/{vehicleID}/surplus";
  let code3 = "curl --header 'Content-Type: application/json' --data '{\"password\": \"\"}' https://chargebot.io/api/1/user/" + apiToken + "/{vehicleID}/plugged_in";
  let code4 = "curl --header 'Content-Type: application/json' --data '{\"password\": \"\"}' https://chargebot.io/api/1/user/" + apiToken + "/{vehicleID}/unplugged";
  let tokenHelp = (
    <Modal show={showTokenHelp} onHide={() => setShowTokenHelp(false)}>
      <Modal.Header closeButton>
        <Modal.Title>How to use your API Token</Modal.Title>
      </Modal.Header>

      <Modal.Body>
        <h5>Update surplus</h5>
        <p>Regularly push your enegery surplus available for charging your vehicle (inverter active power minus consumption) using HTTP POST:</p>
        <CopyBlock text={code1} language="bash" wrapLongLines={true} showLineNumbers={false} />
        <p>Alternatively, you can push your current inverter active power and your household&apos;s consumption separately:</p>
        <CopyBlock text={code2} language="bash" wrapLongLines={true} showLineNumbers={false} />
        <h5 style={{'marginTop': "25px"}}>Update plugged in status</h5>
        <p>If your vehicles gets plugged in:</p>
        <CopyBlock text={code3} language="bash" wrapLongLines={true} showLineNumbers={false} />
        <p>If your vehicles gets unplugged:</p>
        <CopyBlock text={code4} language="bash" wrapLongLines={true} showLineNumbers={false} />
      </Modal.Body>
    </Modal>
  );

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
      <h2 className="pb-3" style={{'marginTop': '50px'}}>API Token</h2>
      <p>An individual API token is required to send your solar surplus and plug in events to chargebot.io.</p>
      {token}
      {tokenHelp}
      <Button variant="primary" onClick={e => setShowTokenHelp(true)}>How to use</Button>
    </>
  );

  return (
    <Container fluid="sm" className="pt-5 container-max-width min-height">
      <h2 className="pb-3">My vehicles</h2>
      <Alert variant='success' dismissible={true} hidden={!showAlertAdded}>Vehicle successfully added to your account.</Alert>
      <Alert variant='success' dismissible={true} hidden={!showAlertRemoved}>Vehicle successfully removed from your account.</Alert>
      {vehicleList}
      {tokenSection}
    </Container>
  )
}
