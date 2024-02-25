'use client'

import React, { useEffect, useState } from "react";
import { checkAuth, getAccessToken, getBaseUrl, saveUserDetails } from "../util";
import { Alert, Container } from "react-bootstrap";
import Loading from "../loading";
import Link from "next/link";

export default function TeslaCallbackPage() {
  const [accessToken, setAccessToken] = useState("")
  const [refreshToken, setRefreshToken] = useState("")
  const [success, setSuccess] = useState(false)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const searchParams = new URLSearchParams(window.location.search);
    const state = searchParams.get("state");
    const code = searchParams.get("code");
    const url = getBaseUrl() + "/api/1/auth/tesla/callback?state=" + state + "&code=" + code;
    const fetchData = async () => {
      await checkAuth();
      fetch(url, {
        method: 'GET',
        headers: {
          Authorization: `Bearer ${getAccessToken()}`,
        }
      })
        .then(res => res.json())
        .then(json => {
          if (json.access_token) {
            setAccessToken(json.access_token);
            setRefreshToken(json.refresh_token);
            saveUserDetails(json.user);
            setSuccess(true);
            setLoading(false);
          }
        })
        .catch(() => {
          setSuccess(false);
          setLoading(false);
        })
        .catch(() => {
          setSuccess(false);
          setLoading(false);
        });
    }
    fetchData();
  }, []);

  if (loading) {
    return (
      <Container fluid="sm" className="pt-5 container-max-width min-height">
        <Loading />
      </Container>
    )
  }

  if (success) {
    return (
      <Container fluid="sm" className="pt-5 container-max-width min-height">
        <Alert variant='success' dismissible={false}>Success! Your Tesla Account has been linked succcessfully.</Alert>
        <p>
          <Link href="/authorized" className="btn btn-link">&lt; Back</Link>
        </p>
      </Container>
    )
  }

  if (!success) {
    return (
      <Container fluid="sm" className="pt-5 container-max-width min-height">
        <Alert variant='danger' dismissible={false}>Something went wrong while linking your Tesla Account. Please try again.</Alert>
        <p>
          <Link href="/authorized" className="btn btn-link">&lt; Back</Link>
        </p>
      </Container>
    )
  }
}
