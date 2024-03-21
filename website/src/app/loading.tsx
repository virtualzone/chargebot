import { Container } from "react-bootstrap";
import { Loader as IconLoad } from 'react-feather';

export default function Loading() {
  return (
    <Container fluid="sm" className="pt-5 container-max-width min-height">
      <p>
        <IconLoad className="feather loader" />
        Loading...
      </p>
    </Container>
  )
}
