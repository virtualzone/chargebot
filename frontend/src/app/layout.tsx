"use client"

import 'bootstrap/dist/css/bootstrap.min.css';
import './global.css'
import { Button, Container, Navbar } from 'react-bootstrap';
import Script from 'next/script';
import Link from 'next/link';
import { getAccessToken } from './util';
import { useEffect, useState } from 'react';

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const [isAuthenticated, setIsAuthenticated] = useState(false)

  useEffect(() => {
    const token = getAccessToken();
    setIsAuthenticated((token !== undefined) && (token !== null) && (token !== ''));
  }, []);

  return (
    <html lang="en" data-bs-theme="auto">
      <head>
        <meta name="charset" content="utf-8" />
        <title>chargebot.io</title>
        <Script type='text/javascript' id='script-auto-dark'>{`
          ;(function () {
            const htmlElement = document.querySelector("html")
            if(htmlElement.getAttribute("data-bs-theme") === 'auto') {
              function updateTheme() {
                document.querySelector("html").setAttribute("data-bs-theme",
                window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light")
              }
              window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', updateTheme)
              updateTheme()
            }
          })()
          var _paq = window._paq = window._paq || [];
          _paq.push(['trackPageView']);
          _paq.push(['enableLinkTracking']);
          (function() {
            var u="https://stats.virtualzone.de/";
            _paq.push(['setTrackerUrl', u+'matomo.php']);
            _paq.push(['setSiteId', '6']);
            var d=document, g=d.createElement('script'), s=d.getElementsByTagName('script')[0];
            g.async=true; g.src=u+'matomo.js'; s.parentNode.insertBefore(g,s);
          })();
      `}
        </Script>
      </head>
      <body>
        <Navbar expand="lg" className="bg-body-tertiary" sticky='top' style={{ 'height': '59px' }}>
          <Container>
            <Navbar.Brand href="/">chargebot.io</Navbar.Brand>
            <Navbar.Text className="justify-content-end">
              <Button variant='link' href='/api/1/auth/init3rdparty' hidden={isAuthenticated}>Sign In</Button>
              <Button variant='link' href='/authorized/' hidden={!isAuthenticated}>My vehicles</Button>
            </Navbar.Text>
          </Container>
        </Navbar>
        {children}
        <footer className="py-3 my-4 border-top d-flex justify-content-center align-items-center">
          <ul className="nav">
            <li className="nav-item"><Link href="/imprint" className="nav-link px-2 text-body-secondary">Imprint</Link></li>
            <li className="nav-item"><Link href="/privacy-policy" className="nav-link px-2 text-body-secondary">Privacy Policy</Link></li>
          </ul>
        </footer>
      </body>
    </html>
  )
}
