'use client'

import React, { useEffect, useState } from "react";
import { getBaseUrl, saveAccessToken, saveRefreshToken, saveUserDetails } from "../util";
import { useRouter } from "next/navigation";
import { Container } from "react-bootstrap";
import Loading from "../loading";

export default function CallbackPage() {
  const [data, setData] = useState(null)
  const router = useRouter()

  useEffect(() => {
    const searchParams = new URLSearchParams(window.location.search);
    const state = searchParams.get("state");
    const code = searchParams.get("code");
    const url = getBaseUrl() + "/api/1/auth/callback?state=" + state + "&code=" + code;
    fetch(url, { method: 'GET' })
      .then(res => res.json())
      .then(json => {
        if (json.access_token) {
          saveAccessToken(json.access_token);
          saveRefreshToken(json.refresh_token);
          saveUserDetails(json.user)
          setData(json);
          router.push('/authorized');
        }
      })
      .catch(e => router.push('/loginfailed'))
      .catch(e => router.push('/loginfailed'));
  }, [router]);

  return (
    <Container fluid="sm" className="pt-5 container-max-width min-height">
      <Loading />
    </Container>
  )
}
