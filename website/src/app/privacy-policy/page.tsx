'use client'

import { Container } from "react-bootstrap"

export default function PagePrivacyPolicy() {
  return (
    <Container fluid="sm" className="pt-5 container-max-width min-height">
      <h2 className="pb-3">Privacy Policy</h2>
      <p>Version: January 22, 2024</p>

      <p>We created this privacy policy in order to inform you about the information we collect, how we use your data and which choices you as a visitor and/or user of chargebot.io have.</p>

      <p>Unfortunately, it's in the nature of things that this policy sounds quite technically. We tried to keep things as simple and clear as possible.</p>

      <h5>Personal data stored</h5>
      <p>The personal information you provide us (such as your name, email address, address or other personal information required in some form) are processed by us together with a timestamp and your IP address only for the stated purpose, stored securely and are not passed on to third parties.</p>

      <p>Thus, we only use your personal information only for the communication with visitors who express this and for providing the offered services and products. We will not pass on your personal data without your consent. This should however not preclude that national authorities can gain access to this data in case of unlawful conduct.</p>

      <p>If you send us personal data by email, we cannot guarantee its secure transmission. We strongly recommend not to send personal data via email without encryption.</p>

      <p>The legislative basis according to article 6 (1) of the DSGVO (lawfulness of processing of personal data) consists of your consent to processing your provided information. You can revoke your consent at any time. An informal email is all it needs. You’ll find out contact information in this website’s imprint.</p>

      <h5>Which personal data we store</h5>
      <p style={{ 'fontWeight': 'bold' }}>On this website</p>

      <p>You can use this website without providing any personal information. If you optionally choose to use functionalities that require the input of personal information, we will only use these for the purpose stated.</p>

      <p style={{ 'fontWeight': 'bold' }}>When signing in</p>

      <p>Using the functionalities of chargebot.io requires you to log in with your Tesla account. The processing of your login details (username, password, and other information) happens on Tesla's servers. We use established standards such as OpenID Connect or OAuth so your sensitive data is processed directly by Tesla's authentication systems. Your credentials are neither stored nor processed by our servers. However, a unique account identifier (such as a User ID or your email address) is always stored on and processed by our systems to identify you on recurring logins.</p>

      <p>Furthermore, we store and process details about your charging preferences as entered by you; and we store and process necessary details about your vehicle(s). This might include the geo location / GPS position of your vehicle in order to determine whether your vehicle is at your home's charging location.</p>

      <h5>Where we store your data</h5>
      <p>Our servers are located in Germany.</p>

      <h5>Your rights according to General Data Protection Regulation (GDPR)</h5>
      <p>According to the regulations of the General Data Protection Regulation (GDPR) you have the following rights:</p>

      <ul>
        <li>Right to have your data corrected (article 16 DSGVO)</li>
        <li>Right to have your data deleted (article 17 DSGVO)</li>
        <li>Right to limit the processing of your data (article 18 DSGVO)</li>
        <li>Right to be notified – Duty regarding the correction, deletion or limitation of your data and its processing (article 19 DSGVO)</li>
        <li>Right to data portability (article 20 DSGVO)</li>
        <li>Right to refuse (article 21 DSGVO)</li>
        <li>Right to be not subject to sole automatic decision making, including profiling (article 22 DSGVO)</li>
      </ul>

      <p>If you think the processing of your data violates the terms of the General Data Protection Regulation (GDPR) or your claims for data protection are violated in any way, you can contact the Federal Commissioner for Data Protection and Freedom of Information in Germany.</p>

      <h5>How long we store your data</h5>
      <p>If you sign up for our services, we will store the data as described above for an indefinite period of time. If you decide to close your account, we will delete all related data directly after the contract has ended. Due to technical reasons, it may be necessary to keep backups after the date the contract ends.</p>

      <h5>Which rights to have regarding your data</h5>
      <p>If you have an account, you can request an export of your personal data from us, including the data you have chosen to share with us. Furthermore, you can request the deletion of all your personal data stored on our systems. This does not include data we have to keep due to administrative, legal or security reasons.</p>

      <h5>Where we send your data</h5>
      <p>We will not share your data with third parties.</p>

      <h5>TLS encryption using HTTPS</h5>
      <p>In both our website and our app, we use HTTPS to transport data securely. (data protection by technical means <a href="https://eur-lex.europa.eu/legal-content/DE/TXT/HTML/?uri=CELEX:32016R0679&from=DE&tid=311177212">article 25 (1) DSGVO</a>). By using TLS (Transport Layer Security), an encryption protocol to securely transport data on the internet, we can protect sensitive data. Most browsers show a lock symbol in your browser when HTTPS is active.</p>

      <h5>Cloudflare</h5>
      <p>We use the “Cloudflare” service provided by Cloudflare Inc., 101 Townsend St., San Francisco, CA 94107, USA. (hereinafter referred to as “Cloudflare”).</p>

      <p>Cloudflare offers a content delivery network with DNS that is available worldwide. As a result, the information transfer that occurs between your browser and our website is technically routed via Cloudflare’s network. This enables Cloudflare to analyze data transactions between your browser and our website and to work as a filter between our servers and potentially malicious data traffic from the Internet. In this context, Cloudflare may also use cookies or other technologies deployed to recognize Internet users, which shall, however, only be used for the herein described purpose.</p>

      <p>The use of Cloudflare is based on our legitimate interest in a provision of our website offerings that is as error free and secure as possible (Art. 6(1)(f) GDPR).</p>

      <p>Data transmission to the US is based on the Standard Contractual Clauses (SCC) of the European Commission. Details can be found here: <a href="https://www.cloudflare.com/privacypolicy/">https://www.cloudflare.com/privacypolicy/</a></p>

      <p>For more information on Cloudflare's security precautions and data privacy policies, please follow this link: <a href="https://www.cloudflare.com/privacypolicy/">https://www.cloudflare.com/privacypolicy/</a></p>

      <h5>Web Analytics</h5>
      <p>For statistical purposes, this website uses Matomo, an open source web analysis tool. Matomo does not transfer any data to servers outside our control. All data is processed and stored anonymised. Matomo is provided by InnoCraft Ltd, 7 Waterloo Quay PO625, 6140 Wellington, New Zealand. You can find out more about the data being processed by Matomo in its privacy policy at <a href="https://matomo.org/privacy-policy/">https://matomo.org/privacy-policy/</a>. If you have any questions regarding the protection of your web analytics data, please contact privacy@matomo.org.</p>

      <p>&nbsp;</p>
      <p>Source: Translation based on the German version created with the <a href="https://www.adsimple.de/datenschutz-generator/">Datenschutz-Generator</a> by AdSimple</p>
    </Container>
  )
}
