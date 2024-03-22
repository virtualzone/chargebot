"use client"

import Image from "next/image";
import { Accordion, Card, Col, Container, Row } from "react-bootstrap";

export default function Home() {
  return (
    <>
      <div className='d-flex justify-content-center align-items-center' style={{ "marginTop": "-54px", "height": "90vh" }}>
        <div className='text-center' data-bs-theme="auto">
          <h1 className='mb-5 display-3'>Charge Your Tesla with Green Energy</h1>
          <h2 className='mb-5'>Maximum Solar Power, Low Grid Prices.</h2>
          <a className='btn btn-danger btn-lg text-white' href='/api/1/auth/login' role='button'>
            <Image src="/tesla-icon.svg" width={24} height={24} alt="" className="me-2" />
            Sign in
          </a>
        </div>
      </div>
      <Container fluid="md" className="pb-5 pt-5">
        <h2 className="pb-3">How it works</h2>
        <Row className="row-eq-height">
          <Col lg={true} className="pb-3">
            <Card className="w-100 h-100" bg='secondary' text='white'>
              <Card.Body>
                <Card.Title>Charge from Solar Power</Card.Title>
                <Card.Text>
                  Make optimum use of your solar power plant: Let your Tesla charge automatically whenever there is enough solar surplus available. The charging capacity is adjusted constantly.
                </Card.Text>
              </Card.Body>
            </Card>
          </Col>
          <Col lg={true} className="pb-3">
            <Card className="w-100 h-100" bg='secondary' text='white'>
              <Card.Body>
                <Card.Title>Utilize low Grid Prices</Card.Title>
                <Card.Text>
                  You're using a grid provider with a dynamic tariff - such as Tibber? Let your Tesla charge automatically when your grid provider's prices are especially low. Choose between different charging strategies: from using very low prices only to ensuring a fully charged car on departure, it's your decision.
                </Card.Text>
              </Card.Body>
            </Card>
          </Col>
          <Col lg={true} className="pb-3">
            <Card className="w-100 h-100" bg='secondary' text='white'>
              <Card.Body>
                <Card.Title>Charging simplified</Card.Title>
                <Card.Text>
                  Just plug in your Tesla and let chargebot.io handle the charging smartly. It works with any wallbox and with any solar power inverter. Works with solar power, dynamic grid prices, or both.
                </Card.Text>
              </Card.Body>
            </Card>
          </Col>
        </Row>
      </Container>
      <Container fluid="md" className="pb-5 pt-5 container-max-width">
        <h2 className="pb-3">FAQ</h2>
        <Accordion flush={true}>
          <Accordion.Item eventKey="0">
            <Accordion.Header>Why do I need to link my Tesla account?</Accordion.Header>
            <Accordion.Body>
              We're using Tesla's Fleet API in order to communicate with your vehicle and to control your Tesla's charging. By linking your Tesla Account with chargebot.io, we show you a Token issued by Tesla which you deploy on your local remote controller node instance. This Token is not stored by chargebot.io.
            </Accordion.Body>
          </Accordion.Item>
          <Accordion.Item eventKey="1">
            <Accordion.Header>Is my data safe?</Accordion.Header>
            <Accordion.Body>
              All relevant data (especially your Tesla Access and Refresh Tokens) is stored only on your local remote controller node within your sovereignty. We neither store nor use your Tesla Tokens within the centralized chargebot.io service.
            </Accordion.Body>
          </Accordion.Item>
          <Accordion.Item eventKey="2">
            <Accordion.Header>What do I need in order to use chargebot.io?</Accordion.Header>
            <Accordion.Body>
              <ul>
                <li>A Tesla vehicle</li>
                <li>A solar power system<br />and/or</li>
                <li>A Tibber contract with Tibber Pulse or an electric meter allowing for dynamic hourly prices</li>
                <li>A home automation system (i.e. Home Assistant, OpenHAB, ioBroker) or another solution which can regularly notify your local chargebot.io node about your solar surplus</li>
              </ul>
            </Accordion.Body>
          </Accordion.Item>
          <Accordion.Item eventKey="3">
            <Accordion.Header>How does it work?</Accordion.Header>
            <Accordion.Body>
              <p>chargebot.io uses the Tesla Fleet API and Tesla Fleet Telemetry in order to control your vehicle's charging process.</p>
              <p>The actual work is done by your local remote controller node. It decides whether there's enough surplus from your solar power plant in order to charge your Tesla. It checks your grid provider for the current prices and starts charging if the prices are below your defined maximum.</p>
              <p>The centralized chargebot.io instance serves as a proxy for your local node's command and forwards them to the Tesla Fleet API. The centralized instance is required as it signs requests from your local node to your Tesla with a private key and forwards incoming Fleet Telemetry data to your local node.</p>
              <p>Only your local node knows and saves your personal Tesla Token. It is neither stored nor used by the centralized chargebot.io instance.</p>
            </Accordion.Body>
          </Accordion.Item>
        </Accordion>
      </Container>
    </>
  )
}
