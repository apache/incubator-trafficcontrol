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

import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';

import { APIService } from './apiservice';

import { Server, Servercheck, checkMap } from '../../models';

@Injectable({providedIn: 'root'})
export class ServerService extends APIService {
	public getServers(): Observable<Array<Server>> {
		let path = `/api/${this.API_VERSION}/servers`;
		return this.get(path).pipe(map(
			r => {
				return r.body.response as Array<Server>;
			}
		));
	}

	public getServerChecks(): Observable<Servercheck[]>
	public getServerChecks(id: number): Observable<Servercheck>
	/**
	 * Fetches server "check" stats from Traffic Ops.
	 * Because the filter is not implemented on the server-side, the returned
	 * Observable<Servercheck> will throw an error if `id` does not exist.
	 * @param id If given, will return only the checks for the server with that ID.
	 * @todo Ideally this filter would be implemented server-side; the data set gets huge.
	 */
	public getServerChecks(id?: number): Observable<Servercheck | Servercheck[]> {
		const path = `/api/${this.API_VERSION}/servercheck`;
		return this.get(path).pipe(map(
			r => {
				const response = r.body.response as Servercheck[];
				if (id) {
					for (const sc of response) {
						if (sc.id === id) {
							sc.checkMap = checkMap;
							return sc;
						}
					}
					throw new ReferenceError(`No server #${id} found in checks response`);
				}
				response.forEach(x=>x.checkMap = checkMap);
				return response;
			}
		));
	}

	constructor(http: HttpClient) {
		super(http);
	}
}
