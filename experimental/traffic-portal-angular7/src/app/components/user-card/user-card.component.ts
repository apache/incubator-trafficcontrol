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
import { Component, Input, ElementRef, OnInit } from '@angular/core';

import { first } from 'rxjs/operators';

import { APIService } from '../../services';
import { Role, User } from '../../models/user';

@Component({
	selector: 'user-card',
	templateUrl: './user-card.component.html',
	styleUrls: ['./user-card.component.scss']
})
export class UserCardComponent implements OnInit {

	@Input() user: User;

	constructor (private readonly api: APIService) { }

	ngOnInit () {
		this.user = this.user as User;
		if (!this.user.roleName) {
			this.api.getRoles(this.user.role).pipe(first()).subscribe(
				(role: Role) => {
					this.user.roleName = role.name;
				}
			);
		}
		// Go emits marshaled JSON date/time structs in a format only Chrome can parse. Because, you know, Google is web standard.
		if (typeof(this.user.lastUpdated) === 'string') {
			this.user.lastUpdated = new Date((this.user.lastUpdated as string).replace('-', '/').replace('-', '/').replace('+', ' GMT+'));
		}
	}

	userHasLocation (): boolean {
		return this.user.city !== null || this.user.stateOrProvince !== null || this.user.country !== null || this.user.postalCode !== null;
	}

	userLocationString (): string | null {
		let ret = '';
		if (this.user.city) {
			ret += this.user.city;
		}
		if (this.user.stateOrProvince) {
			if (ret.length !== 0) {
				ret += ', ';
			}
			ret += this.user.stateOrProvince;
		}
		if (this.user.country) {
			if (ret.length !== 0) {
				ret += ', ';
			}
			ret += this.user.country;
		}
		if (this.user.postalCode) {
			if (ret.length !== 0) {
				ret += ', ';
			}
			ret += this.user.postalCode;
		}

		return ret.length === 0 ? null : ret;
	}
}
