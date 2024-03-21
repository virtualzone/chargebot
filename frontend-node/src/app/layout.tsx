"use client"

import 'bootstrap/dist/css/bootstrap.min.css';
import './global.css'
import { Button, Container, Navbar } from 'react-bootstrap';
import Script from 'next/script';
import Link from 'next/link';
import { getAccessToken } from './util';
import { useEffect, useState } from 'react';
import { BatteryCharging } from 'react-feather';
import { usePathname } from 'next/navigation';

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const pathname = usePathname();

  return (
    <html lang="en" data-bs-theme="auto">
      <head>
        <meta name="charset" content="utf-8" />
        <link rel="icon" type="image/svg+xml" href="/favicon.svg" />
        <link rel="icon" type="image/png" href="/favicon.png" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <meta name="theme-color" content="#212529" />
        <link rel="manifest" href="/manifest.json" />
        <meta name="apple-mobile-web-app-capable" content="yes" />
        <meta name="apple-mobile-web-app-status-bar-style" content="default" />
        <link rel="shortcut icon" href="/favicon-192.png" />
        <link rel="apple-touch-icon" href="/favicon-192.png" />
        <link rel="apple-touch-startup-image" href="/favicon-1024.png" />
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
            <Navbar.Brand href="/"><BatteryCharging /> chargebot.io</Navbar.Brand>
            <Navbar.Text className="justify-content-end">
              <Button variant='link' href='/help/'>Help</Button>
              <Button variant='link' href='/'>My vehicles</Button>
            </Navbar.Text>
          </Container>
        </Navbar>
        {children}
        <footer className="py-3 my-4 border-top d-flex justify-content-center align-items-center">
          <ul className="nav">
            <li className="nav-item"><Link href="/help" className="nav-link px-2 text-body-secondary">Help</Link></li>
            <li className="nav-item"><Link href="/imprint" className="nav-link px-2 text-body-secondary">Imprint</Link></li>
            <li className="nav-item"><Link href="/privacy-policy" className="nav-link px-2 text-body-secondary">Privacy Policy</Link></li>
          </ul>
        </footer>
      </body>
    </html>
  )
}
