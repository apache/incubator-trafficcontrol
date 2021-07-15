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

/** @typedef { import('./CommonGridController').CGC } CGC */

let CommonGridController = function ($scope, $document, $state, userModel, dateUtils) {
    this.entry = null;
    this.quickSearch = "";
    this.pageSize = 100;
    this.showMenu = false;
    this.menuStyle = {
        left: 0,
        top: 0
    };
    this.mouseDownSelectionText = "";

    // Bound Variables
    /** @type string */
    this.title = "";
    /** @type string */
    this.tableName = "";
    /** @type CGC.GridSettings */
    this.options = {};
    /** @type any[] */
    this.columns = [];
    /** @type any[] */
    this.data = [];
    /** @type any[] */
    this.selectedData = [];
    /** @type any */
    this.defaultData = {};
    /** @type CGC.DropDownOption[] */
    this.dropDownOptions = [];
    /** @type CGC.ContextMenuOption[] */
    this.contextMenuOptions = [];
    /** @type CGC.TitleButton */
    this.titleButton = {};
    /** @type CGC.TitleBreadCrumbs */
    this.breadCrumbs = [];

    function HTTPSCellRenderer() {}
    HTTPSCellRenderer.prototype.init = function(params) {
        this.eGui = document.createElement("A");
        this.eGui.href = "https://" + params.value;
        this.eGui.setAttribute("class", "link");
        this.eGui.setAttribute("target", "_blank");
        this.eGui.textContent = params.value;
    };
    HTTPSCellRenderer.prototype.getGui = function() {return this.eGui;};

    // browserify can't handle classes...
    function SSHCellRenderer() {}
    SSHCellRenderer.prototype.init = function(params) {
        this.eGui = document.createElement("A");
        this.eGui.href = "ssh://" + userModel.user.username + "@" + params.value;
        this.eGui.setAttribute("class", "link");
        this.eGui.textContent = params.value;
    };
    SSHCellRenderer.prototype.getGui = function() {return this.eGui;};

    function CheckCellRenderer() {}
    CheckCellRenderer.prototype.init = function(params) {
        this.eGui = document.createElement("I");
        if (params.value === null || params.value === undefined) {
            return;
        }

        this.eGui.setAttribute("aria-hidden", "true");
        this.eGui.setAttribute("title", String(params.value));
        this.eGui.classList.add("fa", "fa-lg");
        if (params.value) {
            this.eGui.classList.add("fa-check");
        } else {
            this.eGui.classList.add("fa-times");
        }
    };
    CheckCellRenderer.prototype.getGui = function() {return this.eGui;};

    function UpdateCellRenderer() {}
    UpdateCellRenderer.prototype.init = function(params) {
        this.eGui = document.createElement("I");

        this.eGui.setAttribute("aria-hidden", "true");
        this.eGui.setAttribute("title", String(params.value));
        this.eGui.classList.add("fa", "fa-lg");
        if (params.value) {
            this.eGui.classList.add("fa-clock-o");
        } else {
            this.eGui.classList.add("fa-check");
        }
    };
    UpdateCellRenderer.prototype.getGui = function() {return this.eGui;};

    function defaultTooltip(params) {
        return params.value;
    }

    function dateCellFormatterRelative(params) {
        return params.value ? dateUtils.getRelativeTime(params.value) : params.value;
    }

    function dateCellFormatterUTC(params) {
        return params.value ? params.value.toUTCString() : params.value;
    }

    this.hasContextItems = function() {
        return this.contextMenuOptions.length > 0;
    };

    this.$onInit = function() {
        let tableName = this.tableName;
        let self = this;

        if (self.defaultData !== undefined) {
            self.entry = self.defaultData;
        }

        for(let i = 0; i < self.columns.length; ++i) {
            if (self.columns[i].filter === "agDateColumnFilter") {
                if (self.columns[i].relative !== undefined && self.columns[i].relative === true) {
                    self.columns[i].tooltipValueGetter = dateCellFormatterRelative;
                    self.columns[i].valueFormatter = dateCellFormatterRelative;
                }
                else {
                    self.columns[i].tooltipValueGetter = dateCellFormatterUTC;
                    self.columns[i].valueFormatter = dateCellFormatterUTC;
                }
            }
        }

        // clicks outside the context menu will hide it
        $document.bind("click", function(e) {
            self.showMenu = false;
            e.stopPropagation();
            $scope.$apply();
        });

        this.gridOptions = {
            components: {
                httpsCellRenderer: HTTPSCellRenderer,
                sshCellRenderer: SSHCellRenderer,
                updateCellRenderer: UpdateCellRenderer,
                checkCellRenderer: CheckCellRenderer,
            },
            columnDefs: self.columns,
            enableCellTextSelection: true,
            suppressMenuHide: true,
            multiSortKey: 'ctrl',
            alwaysShowVerticalScroll: true,
            defaultColDef: {
                filter: true,
                sortable: true,
                resizable: true,
                tooltipValueGetter: defaultTooltip
            },
            rowClassRules: self.options.rowClassRules,
            rowData: self.data,
            pagination: true,
            paginationPageSize: self.pageSize,
            rowBuffer: 0,
            onColumnResized: function() {
                localStorage.setItem(tableName + "_table_columns", JSON.stringify(self.gridOptions.columnApi.getColumnState()));
            },
            colResizeDefault: "shift",
            tooltipShowDelay: 500,
            allowContextMenuWithControlKey: true,
            preventDefaultOnContextMenu: self.hasContextItems(),
            onCellMouseDown: function() {
                self.mouseDownSelectionText = window.getSelection().toString();
            },
            onCellContextMenu: function(params) {
                if (!self.hasContextItems()){
                    return;
                }
                self.showMenu = true;
                self.menuStyle.left = String(params.event.clientX) + "px";
                self.menuStyle.top = String(params.event.clientY) + "px";
                self.menuStyle.bottom = "unset";
                self.menuStyle.right = "unset";
                $scope.$apply();
                const boundingRect = document.getElementById("context-menu").getBoundingClientRect();

                if (boundingRect.bottom > window.innerHeight){
                    self.menuStyle.bottom = String(window.innerHeight - params.event.clientY) + "px";
                    self.menuStyle.top = "unset";
                }
                if (boundingRect.right > window.innerWidth) {
                    self.menuStyle.right = String(window.innerWidth - params.event.clientX) + "px";
                    self.menuStyle.left = "unset";
                }
                self.entry = params.data;
                $scope.$apply();
            },
            onColumnVisible: function(params) {
                if (params.visible){
                    return;
                }
                for (let column of params.columns) {
                    if (column.filterActive) {
                        const filterModel = self.gridOptions.api.getFilterModel();
                        if (column.colId in filterModel) {
                            delete filterModel[column.colId];
                            self.gridOptions.api.setFilterModel(filterModel);
                        }
                    }
                }
            },
            onRowSelected: function() {
                self.selectedData = self.gridOptions.api.getSelectedRows();
                $scope.$apply();
            },
            onSelectionChanged: function() {
                self.selectedData = self.gridOptions.api.getSelectedRows();
                $scope.$apply();
            },
            onRowClicked: function(params) {
                if (params.event.target.classList.contains('link')) {
                    return;
                }
                const selection = window.getSelection().toString();
                if(self.options.onRowClick !== undefined && (selection === "" || selection === $scope.mouseDownSelectionText)) {
                    self.options.onRowClick(params);
                    $scope.$apply();
                }
                $scope.mouseDownSelectionText = "";
            },
            onFirstDataRendered: function() {
                if(self.options.selectRows) {
                    self.gridOptions.rowSelection = self.options.selectRows ? "multiple" : "";
                    self.gridOptions.rowMultiSelectWithClick = self.options.selectRows;
                    self.gridOptions.api.forEachNode(node => {
                        if (node.data[self.options.selectionProperty] === true) {
                            node.setSelected(true, false);
                        }
                    });
                }
                try {
                    const filterState = JSON.parse(localStorage.getItem(tableName + "_table_filters")) || {};
                    self.gridOptions.api.setFilterModel(filterState);
                } catch (e) {
                    console.error("Failure to load stored filter state:", e);
                }

                self.gridOptions.api.addEventListener("filterChanged", function() {
                    localStorage.setItem(tableName + "_table_filters", JSON.stringify(self.gridOptions.api.getFilterModel()));
                });
            },
            onGridReady: function() {
                try {
                    // need to create the show/hide column checkboxes and bind to the current visibility
                    const colstates = JSON.parse(localStorage.getItem(tableName + "_table_columns"));
                    if (colstates) {
                        if (!self.gridOptions.columnApi.setColumnState(colstates)) {
                            console.error("Failed to load stored column state: one or more columns not found");
                        }
                    } else {
                        self.gridOptions.api.sizeColumnsToFit();
                    }
                } catch (e) {
                    console.error("Failure to retrieve required column info from localStorage (key=" + tableName + "_table_columns):", e);
                }

                try {
                    const sortState = JSON.parse(localStorage.getItem(tableName + "_table_sort"));
                    self.gridOptions.api.setSortModel(sortState);
                } catch (e) {
                    console.error("Failure to load stored sort state:", e);
                }

                try {
                    self.quickSearch = localStorage.getItem(tableName + "_quick_search");
                    self.gridOptions.api.setQuickFilter(self.quickSearch);
                } catch (e) {
                    console.error("Failure to load stored quick search:", e);
                }

                try {
                    const ps = localStorage.getItem(tableName + "_page_size");
                    if (ps && ps > 0) {
                        self.pageSize = Number(ps);
                        self.gridOptions.api.paginationSetPageSize(self.pageSize);
                    }
                } catch (e) {
                    console.error("Failure to load stored page size:", e);
                }

                try {
                    const page = parseInt(localStorage.getItem(tableName + "_table_page"));
                    if (page !== undefined && page > 0 && page <= $scope.gridOptions.api.paginationGetTotalPages()-1) {
                        $scope.gridOptions.api.paginationGoToPage(page);
                    }
                } catch (e) {
                    console.error("Failed to load stored page number:", e);
                }

                self.gridOptions.api.addEventListener("sortChanged", function() {
                    localStorage.setItem(tableName + "_table_sort", JSON.stringify(self.gridOptions.api.getSortModel()));
                });

                self.gridOptions.api.addEventListener("columnMoved", function() {
                    localStorage.setItem(tableName + "_table_columns", JSON.stringify(self.gridOptions.columnApi.getColumnState()));
                });

                self.gridOptions.api.addEventListener("columnVisible", function() {
                    self.gridOptions.api.sizeColumnsToFit();
                    try {
                        const colStates = self.gridOptions.columnApi.getColumnState();
                        localStorage.setItem(tableName + "_table_columns", JSON.stringify(colStates));
                    } catch (e) {
                        console.error("Failed to store column defs to local storage:", e);
                    }
                });
            }
        };

    };

    this.exportCSV = function() {
        const params = {
            allColumns: true,
            fileName: this.tablName + ".csv",
        };
        this.gridOptions.api.exportDataAsCsv(params);
    };

    this.toggleVisibility = function(col) {
        const visible = this.gridOptions.columnApi.getColumn(col).isVisible();
        this.gridOptions.columnApi.setColumnVisible(col, !visible);
    };

    this.onQuickSearchChanged = function() {
        this.gridOptions.api.setQuickFilter(this.quickSearch);
        localStorage.setItem(this.title + "_quick_search", this.quickSearch);
    };

    this.onPageSizeChanged = function() {
        const value = Number(this.pageSize);
        this.gridOptions.api.paginationSetPageSize(value);
        localStorage.setItem(this.tableName + "_page_size", value);
    };

    this.clearTableFilters = function() {
        // clear the quick search
        this.quickSearch = '';
        this.onQuickSearchChanged();
        // clear any column filters
        this.gridOptions.api.setFilterModel(null);
    };

    this.contextMenuClick = function(menu, $event) {
        $event.stopPropagation();
        menu.onClick(this.entry);
    };

    this.getHref = function(menu) {
        if (menu.href !== undefined){
            return menu.href;
        }
        return menu.getHref(this.entry);
    };

    this.contextIsDisabled = function(menu) {
        if (menu.isDisabled !== undefined) {
            return menu.isDisabled(this.entry);
        }
        return false;
    };

    this.bcGetText = function (bc) {
        if(bc.text !== undefined){
            return bc.text;
        }
        return bc.getText();
    };

    this.bcHasHref = function(bc) {
        return bc.href !== undefined || bc.getHref !== undefined;
    };

    this.bcGetHref = function(bc) {
        if(bc.href !== undefined) {
            return bc.href;
        }
        return bc.getHref();
    };

    this.getText = function (menu) {
        if (menu.text !== undefined){
            return menu.text;
        }
        return menu.getText(this.entry);
    };

    this.isShown = function (menu) {
        if (menu.shown === undefined){
            return true;
        }
        return menu.shown(this.entry);
    };

    $scope.refresh = function() {
        $state.reload(); // reloads all the resolves for the view
    };
};

angular.module("trafficPortal.table").component("commonGridController", {
    templateUrl: "common/modules/table/agGrid/grid.tpl.html",
    controller: CommonGridController,
    bindings: {
        title: "@",
        tableName: "@",
        options: "<",
        columns: "<",
        data: "<",
        selectedData: "=?",
        dropDownOptions: "<?",
        contextMenuOptions: "<?",
        defaultData: "<?",
        titleButton: "<?",
        breadCrumbs: "<?"
    }
});

CommonGridController.$inject = ["$scope", "$document", "$state", "userModel", "dateUtils"];
module.exports = CommonGridController;
