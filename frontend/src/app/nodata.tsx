import { Container } from "react-bootstrap";


export default function NoData() {
  return (
    <Container fluid="sm" className="pt-5 container-max-width min-height">
      <p>No data found :(</p>
    </Container>
  )
}
