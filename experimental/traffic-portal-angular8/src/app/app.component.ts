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

import { Component } from "@angular/core";
import { Router } from "@angular/router";

import { User } from "./models/user";
import { AuthenticationService } from "./services";

/**
 * AppComponent is the most basic component that contains everything else. This
 * should be kept pretty simple.
 */
@Component({
	selector: "app-root",
	styleUrls: ["./app.component.scss"],
	templateUrl: "./app.component.html"
})
export class AppComponent {
	/** The title of the app. */
	public title = "Traffic Portal";

	/** The currently logged-in user - or 'null' if not logged in. */
	public currentUser: User | null;

	constructor (private readonly router: Router, private readonly auth: AuthenticationService) {
		this.auth.currentUser.subscribe(x => this.currentUser = x);
	}

	/** Logs the current user out. */
	public logout(): void {
		this.auth.logout();
		this.router.navigate(["/login"]);
	}
}
