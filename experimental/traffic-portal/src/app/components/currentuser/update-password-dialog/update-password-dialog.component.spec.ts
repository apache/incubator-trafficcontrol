import { HttpClientModule } from "@angular/common/http";
import { ComponentFixture, TestBed } from "@angular/core/testing";
import { MatDialogModule, MatDialogRef } from "@angular/material/dialog";
import { RouterTestingModule } from "@angular/router/testing";
import { User } from "src/app/models";
import { AuthenticationService } from "src/app/services";
import { UserService } from "src/app/services/api";

import { UpdatePasswordDialogComponent } from "./update-password-dialog.component";

describe("UpdatePasswordDialogComponent", () => {
	let component: UpdatePasswordDialogComponent;
	let fixture: ComponentFixture<UpdatePasswordDialogComponent>;
	let dialogOpen = true;
	let updated = false;

	const mockAPIService = jasmine.createSpyObj(["updateCurrentUser", "getCurrentUser"]);
	mockAPIService.updateCurrentUser.and.returnValue(new Promise(resolve => resolve(true)));
	mockAPIService.getCurrentUser.and.returnValue(new Promise<User>(resolve => resolve({id: -1, newUser: false, username: ""})));

	beforeEach(async () => {
		dialogOpen = true;
		updated = false;
		await TestBed.configureTestingModule({
			declarations: [ UpdatePasswordDialogComponent ],
			imports: [ HttpClientModule, MatDialogModule, RouterTestingModule ],
			providers: [
				{provide: MatDialogRef, useValue: {close: (upd?: true): void => {
					dialogOpen = false;
					updated = upd ?? false;
				}}},
				{provide: UserService, useValue: mockAPIService},
				{provide: AuthenticationService, useValue: {currentUser: {id: -1, newUser: false, username: ""}}}
			]
		}).compileComponents();
	});

	beforeEach(() => {
		fixture = TestBed.createComponent(UpdatePasswordDialogComponent);
		component = fixture.componentInstance;
		fixture.detectChanges();
	});

	it("doesn't allow submitting mismatched passwords", async () => {
		component.password = "password";
		component.confirm = "mismatch";
		await component.submit(new Event("submit"));
		expect(dialogOpen).toBeTrue();
		expect(updated).toBeFalse();
		component.confirmValid.subscribe(
			v => {
				expect(v).toBeTruthy();
			}
		);
		component.confirm = component.password;
		await component.submit(new Event("submit"));
		expect(dialogOpen).toBeFalse();
		expect(updated).toBeTrue();
	});

	it("closes the dialog on cancel", () => {
		component.cancel();
		expect(dialogOpen).toBeFalse();
		expect(updated).toBeFalse();
	});
});
