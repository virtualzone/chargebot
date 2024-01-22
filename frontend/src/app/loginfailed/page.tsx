'use client'

import Link from "next/link"
import { Container } from "react-bootstrap"

export default function Authorized() {
  return (
    <Container fluid="sm" className="vh-100 pt-5">
      <h2 className="pb-3">Login failed</h2>
      <p>Login failed.</p>
      <p><Link href="/">Back to start page</Link></p>
    </Container>
  )
}
