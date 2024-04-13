import { Col, Row } from "react-bootstrap";
import { BatteryCharging as IconCharging, Sun as IconSun, Battery as IconNotCharging, UtilityPole as IconGrid, CarFront as IconCarFront, Unplug as IconUnplugged, PlugZap as IconPluggedIn } from 'lucide-react';

// @ts-ignore: Object is possibly 'null'.
export default function VehicleStatus({ vehicle, state }) {
  function getChargeState(state: any) {
    if (!state.pluggedIn) return <><IconUnplugged /> Disconnected</>;
    if (state.chargingState === 0) return <><IconPluggedIn /> Idle</>;
    if (state.chargingState === 1) return <><IconSun /> Solar-charging at {state.amps} A</>;
    if (state.chargingState === 2) return <><IconGrid /> Grid-charging at {state.amps} A</>;
    return <><IconNotCharging /></>;
  }

  function getSoCIcon(state: any) {
    if (state.chargingState === 0) return <IconNotCharging />;
    if (state.chargingState > 0) return <IconCharging />;
    return <IconNotCharging />;
  }

  if (state === null) state = {};

  return (
    <div>
      <Row>
        <Col style={{ 'flex': '0 0 100px' }}>
          <IconCarFront size={64} />
        </Col>
        <Col style={{ 'flex': '1' }}>
          <h5>{vehicle.display_name}</h5>
          <Row>
            <Col sm={4}>{getSoCIcon(state)} {state.soc !== undefined ? state.soc + ' %' : 'SoC unknown'}</Col>
            <Col sm={8}>{getChargeState(state)}</Col>
          </Row>
        </Col>
      </Row>
    </div>
  )
}
