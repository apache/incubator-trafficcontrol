/*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*    http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*/
import { Component, OnInit } from "@angular/core";
import { faEdit } from "@fortawesome/free-solid-svg-icons";
import { UserService } from "src/app/services/api";

import { User } from "../../models";
import { AuthenticationService } from "../../services";

/**
 * CurrentuserComponent is the controller for the current user's profile page.
 */
@Component({
	selector: "tp-currentuser",
	styleUrls: ["./currentuser.component.scss"],
	templateUrl: "./currentuser.component.html"
})
export class CurrentuserComponent implements OnInit {

	/** The currently logged-in user - or 'null' if not logged-in. */
	public currentUser: User | null = null;
	/** Whether or not the page is in 'edit' mode. */
	private editing = false;
	/** Whether or not the page is in 'edit' mode. */
	public get editMode(): boolean {
		return this.editing;
	}
	/** The icon for the 'edit' button. */
	public editIcon = faEdit;
	/**
	 * The editing copy of the current user - used so that you don't need to
	 * reload the page to see accurate information when the edits are cancelled.
	 */
	public editUser: User | null = null;

	constructor(private readonly auth: AuthenticationService, private readonly api: UserService) {
		this.currentUser = this.auth.currentUser;
	}

	/**
	 * Runs initialization, setting the currently logged-in user from the
	 * authentication service.
	 */
	public ngOnInit(): void {
		if (this.currentUser === null) {
			this.auth.updateCurrentUser().then(
				r => {
					if (r) {
						this.currentUser = this.auth.currentUser;
					}
				}
			);
		}
	}

	/**
	 * Handles when the user clicks on the 'edit' button, making the user's
	 * information editable.
	 */
	public edit(): void {
		if (!this.currentUser) {
			console.error("cannot edit null user");
			return;
		}
		this.editUser = {...this.currentUser};
		this.editing = true;
	}

	/**
	 * Handles when the user click's on the 'cancel' button to cancel edits to
	 * the user's information.
	 */
	public cancelEdit(): void {
		// It's impossible to be in edit mode with a null user
		this.editUser = {...(this.currentUser as User)};
		this.editing = false;
	}

	/**
	 * Handles submission of the user edit form.
	 *
	 * @param e The form submittal event.
	 */
	public submitEdit(e: Event): void {
		e.preventDefault();
		e.stopPropagation();

		this.api.updateCurrentUser(this.editUser as User).then(
			success => {
				if (success) {
					this.auth.updateCurrentUser().then(
						updated => {
							if (!updated) {
								console.warn("Failed to fetch current user after successful update");
							}
							this.currentUser = this.auth.currentUser;
							this.cancelEdit();
						}
					);
				} else {
					console.warn("Editing the current user failed");
					this.cancelEdit();
				}
			}
		);
	}
}
