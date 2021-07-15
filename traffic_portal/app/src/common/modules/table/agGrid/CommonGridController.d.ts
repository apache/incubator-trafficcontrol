/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

import {ColDef, GridOptions, RowClickedEvent} from "ag-grid-community/main";

export namespace CGC {
    export enum OptionType {
        Seperator = 0,
        Button = 1,
        Ancor = 2
    }

    export interface GridSettings extends GridOptions {
        selectRows?: boolean;
        selectionProperty?: string;
        refreshable?: boolean;

        onRowClick(row: RowClickedEvent): void;
    }

    export interface ColumnDefinition extends ColDef {

    }

    export interface TitleBreadCrumbs {
        // If href is undefined, getHref is called
        href?: string;
        getHref?(): string;

        // If text is undefined, getText is called
        text?: string;
        getText(): string;
    }

    export interface CommonOption {
        type: OptionType;
        name: string;
        newTab: boolean;

        onClick(row: any): void;
        isDisabled(row: any): boolean;
        shown(row: any): void;

        // If href is undefined, getHref is called
        href?: string;
        getHref(row: any): string;

        // If text is undefined, getText is called
        text?: string;
        getText(row: any): string;
    }

    export interface TitleButton {
        onClick(): void;
        getText(): string;
    }

    export interface DropDownOption extends CommonOption {
    }

    export interface ContextMenuOption extends CommonOption{
    }

}
