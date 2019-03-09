/*
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package com.comcast.cdn.traffic_control.traffic_router.secure;

import com.comcast.cdn.traffic_control.traffic_router.shared.CertificateData;
import org.apache.log4j.Logger;
import org.bouncycastle.jcajce.provider.asymmetric.rsa.BCRSAPrivateCrtKey;
import sun.security.rsa.RSAPrivateCrtKeyImpl;
import sun.security.rsa.RSAPublicKeyImpl;

import java.math.BigInteger;
import java.security.PrivateKey;
import java.security.PublicKey;
import java.security.cert.CertificateExpiredException;
import java.security.cert.CertificateNotYetValidException;
import java.security.cert.X509Certificate;
import java.util.ArrayList;
import java.util.List;

public class CertificateDataConverter {
	private static final Logger log = Logger.getLogger(CertificateDataConverter.class);

	private PrivateKeyDecoder privateKeyDecoder = new PrivateKeyDecoder();
	private CertificateDecoder certificateDecoder = new CertificateDecoder();

	@SuppressWarnings({"PMD.CyclomaticComplexity"})
	public HandshakeData toHandshakeData(final CertificateData certificateData) {
		try {
			final PrivateKey privateKey = privateKeyDecoder.decode(certificateData.getCertificate().getKey());
			final List<String> encodedCertificates = certificateDecoder.doubleDecode(certificateData.getCertificate().getCrt());

			final List<X509Certificate> x509Chain = new ArrayList<>();
			boolean hostMatch = false;
			boolean modMatch = false;
			for ( final String encodedCertificate : encodedCertificates) {
				final X509Certificate certificate = certificateDecoder.toCertificate(encodedCertificate);
				certificate.checkValidity();
				if (!hostMatch && verifySubject(certificate, certificateData.alias())) {
					hostMatch = true;
				}
				if (!modMatch && verifyModulus(privateKey, certificate)) {
					modMatch = true;
				}
				x509Chain.add(certificate);
			}
			if ( hostMatch && modMatch) {
				return new HandshakeData(certificateData.getDeliveryservice(), certificateData.getHostname(),
						x509Chain.toArray(new X509Certificate[x509Chain.size()]), privateKey);
			}
			else if(!hostMatch){
				log.warn("Service name doesn't match the subject of the certificate = "+certificateData.getHostname());
			}
			else if (!modMatch) {
				log.error("Modulus not == for host: "+certificateData.getHostname());
			}

		} catch ( CertificateNotYetValidException er) {
			log.error("Failed to convert certificate data for delivery service = " + certificateData.getHostname()
							+ ", because the certificate is not valid yet. ");
		} catch (CertificateExpiredException ex ) {
			log.error("Failed to convert certificate data for delivery service = " + certificateData.getHostname()
					+ ", because the certificate has expired. ");
		} catch (Exception e) {
			log.error("Failed to convert certificate data (delivery service = " + certificateData.getDeliveryservice()
					+ ", hostname = " + certificateData.getHostname() + ") from traffic ops to handshake data! "
					+ e.getClass().getSimpleName() + ": " + e.getMessage(), e);
		}
		return null;
	}

	public boolean verifySubject(final X509Certificate certificate, final String hostAlias ) {
		final String host = certificate.getSubjectDN().getName();
		if (hostCompare(hostAlias,host)) {
			return true;
		}

		try {
			// This approach is probably the only one that is JDK independent
			if (certificate.getSubjectAlternativeNames() != null) {
				for (final List<?> altName : certificate.getSubjectAlternativeNames()) {
					if (hostCompare(hostAlias, (String) altName.get(1))) {
						return true;
					}
				}
			}
		}
		catch (Exception e) {
			log.error("Encountered an error while validating the certificate subject for service: "+hostAlias+", " +
					"error: "+e.getClass().getSimpleName()+": " + e.getMessage(), e);
			return false;
		}

		return false;
	}

	private boolean hostCompare(final String hostAlias, final String subject) {
		if (hostAlias.contains(subject) || subject.contains(hostAlias)) {
			return true;
		}
		final String[] chopped = subject.split("CN=", 2);
		if (chopped != null && chopped.length > 1) {
			String chop = chopped[1];
			chop = chop.replaceFirst("\\*\\.", ".");
			chop = chop.split(",", 2)[0];
			if (chop.length()>0 && (hostAlias.contains(chop) || chop.contains(hostAlias))) {
				return true;
			}
		}
		return false;
	}

	public boolean verifyModulus(final PrivateKey privateKey, final X509Certificate certificate) {
		BigInteger privModulus = null;
		if (privateKey instanceof BCRSAPrivateCrtKey) {
			privModulus = ((BCRSAPrivateCrtKey) privateKey).getModulus();
		} else if (privateKey instanceof RSAPrivateCrtKeyImpl) {
			privModulus = ((RSAPrivateCrtKeyImpl) privateKey).getModulus();
		}
		BigInteger pubModulus = null;
		final PublicKey publicKey = certificate.getPublicKey();
		if ((publicKey instanceof RSAPublicKeyImpl)) {
			pubModulus = ((RSAPublicKeyImpl) publicKey).getModulus();
		} else {
			final String[] keyparts = publicKey.toString().split(System.getProperty("line.separator"));
			for (final String part : keyparts) {
				final int start = part.indexOf("modulus: ") + 9;
				if (start < 9) {
					continue;
				} else {
					pubModulus = new BigInteger(part.substring(start));
					break;
				}
			}
		}
		if (privModulus.equals(pubModulus)) {
			return true;
		}
		return false;
	}

	public PrivateKeyDecoder getPrivateKeyDecoder() {
		return privateKeyDecoder;
	}

	public void setPrivateKeyDecoder(final PrivateKeyDecoder privateKeyDecoder) {
		this.privateKeyDecoder = privateKeyDecoder;
	}

	public CertificateDecoder getCertificateDecoder() {
		return certificateDecoder;
	}

	public void setCertificateDecoder(final CertificateDecoder certificateDecoder) {
		this.certificateDecoder = certificateDecoder;
	}
}
