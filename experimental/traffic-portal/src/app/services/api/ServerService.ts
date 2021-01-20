/*
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

import { HttpClient } from "@angular/common/http";
import { Injectable } from "@angular/core";

import { Observable } from "rxjs";
import { map } from "rxjs/operators";

import { Server, Servercheck } from "../../models";

import { APIService } from "./apiservice";

/**
 * ServerService exposes API functionality related to Servers.
 */
@Injectable({providedIn: "root"})
export class ServerService extends APIService {

	/**
	 * Injects the Angular HTTP client service into the parent constructor.
	 *
	 * @param http The Angular HTTP client service.
	 */
	constructor(http: HttpClient) {
		super(http);
	}

	/**
	 * Retrieves all servers.
	 *
	 * @returns An Observable that will emit the servers.
	 */
	public getServers(): Observable<Array<Server>> {
		const path = `/api/${this.apiVersion}/servers`;
		return this.get(path).pipe(map(
			r => (r.body as {response: Array<Server>}).response.map(
				s => {
					if (s.lastUpdated) {
						// Our dates are actually strings since JSON doesn't provide a native date type.
						// TODO: rework to use an interceptor
						const dateStr = (s.lastUpdated as unknown) as string;
						s.lastUpdated = new Date(dateStr.replace(" ", "T").replace(/\+00$/, "Z"));
					}
					return s;
				}
			)
		));
	}

	public getServerChecks(): Observable<Servercheck[]>;
	public getServerChecks(id: number): Observable<Servercheck>;
	/**
	 * Fetches server "check" stats from Traffic Ops.
	 * Because the filter is not implemented on the server-side, the returned
	 * Observable<Servercheck> will throw an error if `id` does not exist.
	 *
	 * @param id If given, will return only the checks for the server with that ID.
	 * @todo Ideally this filter would be implemented server-side; the data set gets huge.
	 * @returns An observable that emits Serverchecks - or a single Servercheck if ID was given.
	 */
	public getServerChecks(id?: number): Observable<Servercheck | Servercheck[]> {
		const path = `/api/${this.apiVersion}/servercheck`;
		return this.get(path).pipe(map(
			r => {
				const response = (r.body as {response: Array<Servercheck>}).response;
				if (id) {
					for (const sc of response) {
						if (sc.id === id) {
							return sc;
						}
					}
					throw new Error(`No server #${id} found in checks response`);
				}
				return response;
			}
		));
	}
}
