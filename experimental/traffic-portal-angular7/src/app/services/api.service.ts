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
import { HttpClient, HttpHeaders, HttpResponse, HttpParams } from '@angular/common/http';
import { BehaviorSubject, Observable, throwError } from 'rxjs';
import { map, first, catchError } from 'rxjs/operators';

import { CDN } from '../models/cdn';
import { DeliveryService } from '../models/deliveryservice';
import { Type } from '../models/type';
import { Role, User } from '../models/user';

@Injectable({ providedIn: 'root' })
/**
 * The APIService provides access to the Traffic Ops API. Its methods should be kept API-version
 * agnostic (from the caller's perspective), and always return `Observable`s.
*/
export class APIService {
	public API_VERSION = '1.4';

	// private cookies: string;

	constructor (private readonly http: HttpClient) {

	}

	private delete (path: string, data?: any): Observable<HttpResponse<any>> {
		return this.do('delete', path, data);
	}
	private get (path: string, data?: any): Observable<HttpResponse<any>> {
		return this.do('get', path, data);
	}
	private head (path: string, data?: any): Observable<HttpResponse<any>> {
		return this.do('head', path, data);
	}
	private options (path: string, data?: any): Observable<HttpResponse<any>> {
		return this.do('options', path, data);
	}
	private patch (path: string, data?: any): Observable<HttpResponse<any>> {
		return this.do('patch', path, data);
	}
	private post (path: string, data?: any): Observable<HttpResponse<any>> {
		return this.do('post', path, data);
	}
	private push (path: string, data?: any): Observable<HttpResponse<any>> {
		return this.do('push', path, data);
	}

	private do (method: string, path: string, data?: Object): Observable<HttpResponse<any>> {

		/* tslint:disable */
		const options = {headers: new HttpHeaders({'Content-Type': 'application/json'}),
		                 observe: 'response' as 'response',
		                 responseType: 'json' as 'json',
		                 body: data};
		/* tslint:enable */
		return this.http.request(method, path, options).pipe(map((response) => {
			// TODO pass alerts to the alert service
			// (TODO create the alert service)
			return response as HttpResponse<any>;
		}));
	}

	/**
	 * Performs authentication with the Traffic Ops server.
	 * @param u The username to be used for authentication
	 * @param p The password of user `u`
	 * @returns An observable that will emit the entire HTTP response
	*/
	public login (u: string, p: string): Observable<HttpResponse<any>> {
		const path = '/api/' + this.API_VERSION + '/user/login';
		return this.post(path, {u, p});
	}

	/**
	 * Fetches the current user from Traffic Ops
	 * @returns An observable that will emit a `User` object representing the current user.
	*/
	public getCurrentUser (): Observable<User> {
		const path = '/api/' + this.API_VERSION + '/user/current';
		return this.get(path).pipe(map(
			r => {
				return r.body.response as User;
			}
		));
	}

	/**
	 * Gets a list of all visible Delivery Services
	 * @returns An observable that will emit an array of `DeliveryService` objects.
	*/
	public getDeliveryServices (): Observable<DeliveryService[]> {
		const path = '/api/' + this.API_VERSION + '/deliveryservices';
		return this.get(path).pipe(map(
			r => {
				return r.body.response as DeliveryService[];
			}
		));
	}

	/**
	 * Creates a new Delivery Service
	 * @param ds The new Delivery Service object
	 * @returns An Observable that will emit a boolean value indicating the success of the operation
	*/
	public createDeliveryService (ds: DeliveryService): Observable<boolean> {
		const path = '/api/' + this.API_VERSION + '/deliveryservices';
		return this.post(path, ds).pipe(map(
			r => {
				return true;
			},
			e => {
				return false;
			}
		));
	}

	/**
	 * Retrieves capacity statistics for the Delivery Service identified by a given, unique,
	 * integral value.
	 * @param d The integral, unique identifier of a Delivery Service
	 * @returrns An Observable that emits an object that hopefully has the right keys to represent capacity.
	*/
	public getDSCapacity (d: number): Observable<any> {
		const path = '/api/' + this.API_VERSION + '/deliveryservices/' + String(d) + '/capacity';
		return this.get(path).pipe(map(
			r => {
				return r.body.response;
			}
		));
	}

	/**
	 * Retrieves the Cache Group health of a Delivery Service identified by a given, unique,
	 * integral value.
	 * @param d The integral, unique identifier of a Delivery Service
	 * @returns An Observable that emits a response from the health endpoint
	*/
	public getDSHealth (d: number): Observable<any> {
		const path = '/api/' + this.API_VERSION + '/deliveryservices/' + String(d) + '/health';
		return this.get(path).pipe(map(
			r => {
				return r.body.response;
			}
		));
	}

	/**
	 * Retrieves Delivery Service throughput statistics for a given time period, averaged over a given
	 * interval.
	 * @param d The `xml_id` of a Delivery Service
	 * @param start A date/time from which to start data collection
	 * @param end A date/time at which to end data collection
	 * @param interval A unit-suffixed interval over which data will be "binned"
	 * @param useMids Collect data regarding Mid-tier cache servers rather than Edge-tier cache servers
	 * @returns An Observable that will emit an Array of datapoint Arrays (length 2 containing a date string and data value)
	*//* tslint:disable */
	public getDSKBPS (d: string,
	                  start: Date,
	                  end: Date,
	                  interval: string,
	                  useMids?: boolean): Observable<Array<Array<any>>> {
		/* tslint:enable */
		let path = '/api/' + this.API_VERSION + '/deliveryservice_stats?metricType=kbps';
		path += '&interval=' + interval;
		path += '&deliveryServiceName=' + d;
		path += '&startDate=' + start.toISOString();
		path += '&endDate=' + end.toISOString();
		path += '&serverType=' + (useMids ? 'mid' : 'edge');
		return this.get(path).pipe(map(
			r => {
				if (r && r.body && r.body.response && r.body.response.series) {
					return r.body.response.series.values;
				}
				return null;
			}
		));
	}

	/**
	 * Gets an array of all users in Traffic Ops
	 * @returns An Observable that will emit an Array of User objects.
	*/
	public getUsers (): Observable<Array<User>> {
		const path = '/api/' + this.API_VERSION + '/users';
		return this.get(path).pipe(map(
			r => {
				return r.body.response as Array<User>;
			}
		));
	}

	public getRoles (id: number): Observable<Role>;
	public getRoles (name: string): Observable<Role>;
	public getRoles (): Observable<Array<Role>>;
	/**
	 * Fetches one or all Roles from Traffic Ops
	 * @param name Optionally, the name of a single Role which will be fetched
	 * @param id Optionally, the integral, unique identifier of a single Role which will be fetched
	 * @throws {TypeError} When called with an improper argument.
	 * @returns an Observable that will emit either an Arry of Roles, or a single Role, depending on whether
	 *	`name`/`id` was passed
	 * (In the event that `name`/`id` is given but does not match any Role, `null` will be emitted)
	*/
	public getRoles (nameOrID?: string | number) {
		const path = '/api/' + this.API_VERSION + '/roles';
		if (nameOrID) {
			switch (typeof nameOrID) {
				case 'string':
					return this.get(path + '?name=' + nameOrID).pipe(map(
						r => {
							for (const role of (r.body.response as Array<Role>)) {
								if (role.name === nameOrID) {
									return role;
								}
							}
							return null;
						}
					));
					break;
				case 'number':
					return this.get(path + '?id=' + nameOrID.toString()).pipe(map(
						r => {
							for (const role of (r.body.response as Array<Role>)) {
								if (role.id === nameOrID) {
									return role;
								}
							}
						}
					));
					break;
				default:
					throw new TypeError("expected a name or ID, got '" + typeof(name) + "'");
					break;
			}
		}
		return this.get(path).pipe(map(
			r => {
				return r.body.response as Array<Role>;
			}
		));
	}

	/**
	 * Gets one or all Types from Traffic Ops
	 * @param name Optionally, the name of a single Type which will be returned
	 * @returns An Observable that will emit either a Map of Type names to full Type objects, or a single Type, depending on whether
	 * 	`name` was passed
	 * (In the event that `name` is given but does not match any Type, `null` will be emitted)
	*/
	public getTypes (name?: string): Observable<Map<string, Type> | Type> {
		const path = '/api/' + this.API_VERSION + '/types';
		if (name) {
			return this.get(path + '?name=' + name).pipe(map(
				r => {
					for (const t of (r.body.response as Array<Type>)) {
						if (t.name === name) {
							return t;
						}
					}
					return null;
				}
			));
		}
		return this.get(path).pipe(map(
			r => {
				const ret = new Map<string, Type>();
				for (const t of (r.body.response as Array<Type>)) {
					ret.set(t.name, t);
				}
				return ret;
			}
		));
	}

	/**
	 * Gets one or all CDNs from Traffic Ops
	 * @param id The integral, unique identifier of a single CDN to be returned
	 * @returns An Observable that will emit either a Map of CDN names to full CDN objects, or a single CDN, depending on whether `id` was
	 * 	passed.
	 * (In the event that `id` is passed but does not match any CDN, `null` will be emitted)
	*/
	public getCDNs (id?: number): Observable<Map<string, CDN> | CDN> {
		const path = '/api/' + this.API_VERSION + '/cdns';
		if (id) {
			return this.get(path + '?id=' + String(id)).pipe(map(
				r => {
					for (const c of (r.body.response as Array<CDN>)) {
						if (c.id === id) {
							return c;
						}
					}
				}
			));
		}
		return this.get(path).pipe(map(
			r => {
				const ret = new Map<string, CDN>();
				for (const c of (r.body.response as Array<CDN>)) {
					ret.set(c.name, c);
				}
				return ret;
			}
		));
	}
}
