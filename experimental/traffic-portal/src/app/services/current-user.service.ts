import { Injectable } from "@angular/core";
import { Router } from "@angular/router";
import { Capability, User } from "../models";

/**
 * This service keeps track of the currently authenticated user.
 *
 * This needs to be done separately from the AuthenticationService's
 * methods, because those depend on the API services and the API services use
 * an implicitly injected ErrorInterceptor which clears the authenticated user
 * value when it hits a 401 error - so that would be a circular dependency.
 */
@Injectable({
	providedIn: "root"
})
export class CurrentUserService {
	/** The currently authenticated user - or `null` if not authenticated. */
	private user: User | null = null;
	/** The Permissions afforded to the currently authenticated user. */
	private caps = new Set<string>();

	/** The currently authenticated user - or `null` if not authenticated. */
	public get currentUser(): User | null {
		return this.user;
	}
	/** The Permissions afforded to the currently authenticated user. */
	public get capabilities(): Set<string> {
		return this.caps;
	}

	/** Whether or not the user is authenticated. */
	public get loggedIn(): boolean {
		return this.currentUser !== null;
	}

	constructor(private readonly router: Router) {}

	/**
	 * Sets the currently authenticated user.
	 *
	 * @param u The new user who has been authenticated.
	 * @param caps The newly authenticated user's Permissions.
	 */
	public setUser(u: User, caps: Set<string> | Array<Capability>): void {
		this.user = u;
		this.caps = caps instanceof Array ? new Set(caps.map(c=>c.name)) : caps;
	}

	/**
	 * Checks if the user has a given Permission.
	 *
	 * @param perm The Permission in question.
	 * @returns `true` if the user has the Permission `perm`, `false` otherwise.
	 */
	public hasPermission(perm: string): boolean {
		return this.user ? this.caps.has(perm) : false;
	}

	/** Clears authentication data associated with the current user. */
	public logout(): void {
		this.user = null;
		this.caps.clear();
		this.router.navigate(["/login"]);
	}
}
