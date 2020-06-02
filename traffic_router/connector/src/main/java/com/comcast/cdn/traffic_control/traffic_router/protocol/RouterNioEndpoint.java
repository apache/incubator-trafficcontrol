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

package com.comcast.cdn.traffic_control.traffic_router.protocol;

import com.comcast.cdn.traffic_control.traffic_router.secure.CertificateRegistry;
import com.comcast.cdn.traffic_control.traffic_router.secure.HandshakeData;
import com.comcast.cdn.traffic_control.traffic_router.secure.KeyManager;
import org.apache.log4j.Logger;
import org.apache.tomcat.jni.SSL;
import org.apache.tomcat.util.net.NioChannel;
import org.apache.tomcat.util.net.NioEndpoint;
import org.apache.tomcat.util.net.SSLHostConfig;
import org.apache.tomcat.util.net.SSLHostConfigCertificate;
import org.apache.tomcat.util.net.SocketEvent;
import org.apache.tomcat.util.net.SocketProcessorBase;
import org.apache.tomcat.util.net.SocketWrapperBase;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.Set;

public class RouterNioEndpoint extends NioEndpoint {
	private static final Logger LOGGER = Logger.getLogger(RouterNioEndpoint.class);
	private String protocols;

	// Grabs the aliases from our custom certificate registry, creates a sslHostConfig for them
	// and adds the newly created config to the list of sslHostConfigs.  We also remove the default config
	// since it won't be found in our registry.  This allows OpenSSL to start successfully and serve our
	// certificates.  When we are done we call the parent classes initialiseSsl.
	@SuppressWarnings({"PMD.SignatureDeclareThrowsException"})
	@Override
	protected void initialiseSsl() throws Exception {
		if (isSSLEnabled()) {
			destroySsl();
			sslHostConfigs.clear();
			final KeyManager keyManager = new KeyManager();
			final CertificateRegistry certificateRegistry = keyManager.getCertificateRegistry();
			replaceSSLHosts(certificateRegistry.getHandshakeData());

			//Now let initialiseSsl do it's thing.
			super.initialiseSsl();
			certificateRegistry.setEndPoint(this);
		}
	}

	synchronized private List<String> replaceSSLHosts(final Map<String, HandshakeData> sslHostsData) {
		final Set<String> aliases = sslHostsData.keySet();
		String lastHostName = "";
		final List<String> failedUpdates = new ArrayList<>();

		for (final String alias : aliases) {
			final SSLHostConfig sslHostConfig = new SSLHostConfig();
			final SSLHostConfigCertificate cert = new SSLHostConfigCertificate(sslHostConfig, SSLHostConfigCertificate.Type.RSA);
			sslHostConfig.setHostName(sslHostsData.get(alias).getHostname());
			cert.setCertificateKeyAlias(alias);
			sslHostConfig.addCertificate(cert);
			sslHostConfig.setCertificateKeyAlias(alias);
			sslHostConfig.setProtocols(protocols != null ? protocols : "all");
			// sslHostConfig.setConfigType(getSslConfigType());
			sslHostConfig.setCertificateVerification("none");
			LOGGER.info("sslHostConfig: "+sslHostConfig.getHostName() + " " + sslHostConfig.getTruststoreAlgorithm());

		if (!sslHostConfig.getHostName().equals(lastHostName)){
			try{
				addSslHostConfig(sslHostConfig, true);
			} catch (Exception fubar){
				LOGGER.error("In RouterNioEndpoint.replaceSSLHosts, sslHostConfig and certs did not get replaced " +
				  "for host: " + sslHostConfig.getHostName() + ", because of execption - " + fubar.toString());
				failedUpdates.add(alias);
			}
			lastHostName = sslHostConfig.getHostName();
		}

			if (CertificateRegistry.DEFAULT_SSL_KEY.equals(alias) && !failedUpdates.contains(alias)){
				// One of the configs must be set as the default
				setDefaultSSLHostConfigName(sslHostConfig.getHostName());
			}
		}
		return failedUpdates;
	}

	synchronized public List<String> reloadSSLHosts(final Map<String, HandshakeData> cr) {
		final List<String> failedUpdates = replaceSSLHosts(cr);
		if (!failedUpdates.isEmpty()) {
			failedUpdates.forEach(alias-> {
				cr.remove(alias);
			});
		}

		final List<String> failedContextUpdates = new ArrayList<>();
		for (final String alias : cr.keySet()) {
			try{
				final HandshakeData data = cr.get(alias);
				final SSLHostConfig sslHostConfig = sslHostConfigs.get(data.getHostname());
				// sslHostConfig.setConfigType(getSslConfigType());
				createSSLContext(sslHostConfig);
			}
			catch (Exception rfubar) {
				LOGGER.error("In RouterNioEndpoint could not create new SSLContext for cert " + alias +
						" because of exception: "+rfubar.toString());
				failedContextUpdates.add(alias);
			}
		}

		if (!failedContextUpdates.isEmpty()) {
			failedUpdates.addAll(failedContextUpdates);
		}

		return failedUpdates;
	}

	@Override
	protected SSLHostConfig getSSLHostConfig(final String sniHostName){
		return super.getSSLHostConfig(sniHostName.toLowerCase());
	}

	@Override
	protected SocketProcessorBase<NioChannel> createSocketProcessor(
			final SocketWrapperBase<NioChannel> socketWrapper, final SocketEvent event){
		return new RouterSocketProcessor(socketWrapper, event);
	}

	/**
	 * This class is the equivalent of the Worker, but will simply use in an
	 * external Executor thread pool.
	 */
	protected class RouterSocketProcessor extends SocketProcessor {

		public RouterSocketProcessor(final SocketWrapperBase<NioChannel> socketWrapper, final SocketEvent event){
			super(socketWrapper, event);
		}

		/* This override has been added as a temporary hack to resolve an issue in Tomcat.
		Once the issue has been corrected in Tomcat then this can be removed. The
		'SSL.getLastErrorNumber()' removes an unwanted error condition from the error stack
		in those cases where some error condition has caused the socket to get closed and
		then the processor was put back on the processor stack for reuse in a future connection.
		*/
		@Override
		protected void doRun(){
			final SocketWrapperBase<NioChannel> localWrapper = socketWrapper;
			final NioChannel socket = localWrapper.getSocket();
			super.doRun();
			if (!socket.isOpen()){
				SSL.getLastErrorNumber();
			}
		}
	}

	public String getProtocols() {
		return protocols;
	}

	public void setProtocols(final String protocols) {
		this.protocols = protocols;
	}

}
