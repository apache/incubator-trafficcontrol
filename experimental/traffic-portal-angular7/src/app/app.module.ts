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

/**
 * This file contains the definition for the entire app. Its syntax is a bit arcane, but hopefully
 * by copy/pasting any novice can add a new component.
*/

import { BrowserModule } from '@angular/platform-browser';
import { ReactiveFormsModule, FormsModule } from '@angular/forms';
import { HttpClientModule, HTTP_INTERCEPTORS } from '@angular/common/http';
import { NgModule } from '@angular/core';
import { Routes, RouterModule } from '@angular/router';

// Routing
import { AppRoutingModule } from './app-routing.module';
import { ErrorInterceptor } from './interceptor/error.interceptor';
import { AlertInterceptor } from './interceptor/alerts.interceptor';

// Components
import { AppComponent } from './app.component';
import { AlertComponent } from './components/alert/alert.component';
import { DsCardComponent } from './components/ds-card/ds-card.component';
import { LoginComponent } from './components/login/login.component';
import { DashboardComponent } from './components/dashboard/dashboard.component';
import { UsersComponent } from './components/users/users.component';
import { NewDeliveryServiceComponent } from './components/new-delivery-service/new-delivery-service.component';
import { TpHeaderComponent } from './components/tp-header/tp-header.component';

/**
 * This is the list of available, distinct URLs, with the leading path separator omitted. Each
 * element should contain a `path` key for the path value, a component which will be inserted at the
 * `<router-outlet>` when the user navigates to `path`, and an optional `canActivate` key which
 * should be a list of services that implement the `CanActivate` interface.
*/
const appRoutes: Routes = [
	{ path: '', component: DashboardComponent },
	{ path: 'login', component: LoginComponent },
	{ path: 'users', component: UsersComponent},
	{ path: 'new.Delivery.Service', component: NewDeliveryServiceComponent}
];

@NgModule({
	declarations: [
		AppComponent,
		LoginComponent,
		DashboardComponent,
		DsCardComponent,
		AlertComponent,
		UsersComponent,
		NewDeliveryServiceComponent,
		TpHeaderComponent,
	],
	imports: [
		BrowserModule.withServerTransition({ appId: 'serverApp' }),
		RouterModule.forRoot(appRoutes),
		AppRoutingModule,
		HttpClientModule,
		ReactiveFormsModule,
		FormsModule
	],
	providers: [
		{provide: HTTP_INTERCEPTORS, useClass: ErrorInterceptor, multi: true},
		{provide: HTTP_INTERCEPTORS, useClass: AlertInterceptor, multi: true}
	],
	bootstrap: [AppComponent]
})
export class AppModule { }
