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

module.exports = {
    'shared-libs-prod': {
        entry: './<%= globalConfig.srcdir %>/scripts/shared-libs.js',
        compile: './<%= globalConfig.resourcesdir %>/assets/js/shared-libs.js',
        options: {
            expose: {
                files: [
                    {
                        cwd: '<%= globalConfig.app %>/bower_components/',
                        src:
                            [
                                'angular-bootstrap/ui-bootstrap.min.js',
                                'angular-bootstrap/ui-bootstrap-tpls.min.js',
                                'angular-loading-bar/build/loading-bar.min.js',
                                'angular-resource/angular-resource.min.js',
                                'angular-route/angular-route.min.js',
                                'angular-sanitize/angular-sanitize.min.js',
                                'angular-ui-router/release/angular-ui-router.min.js',
                                'bootstrap-sass-official/assets/javascripts/bootstrap.min.js',
                                'jquery/jquery.min.js',
                            ]
                    },
                    {
                        cwd: "<%= globalConfig.importdir %>",
                        src: [ "angular/angular.min.js" ]
                    },
                    {
                        cwd: "<%= globalConfig.importdir %>",
                        src: [ "angular-animate/angular-animate.min.js" ]
                    },
                    {
                        cwd: "<%= globalConfig.importdir %>",
                        src: [ "angular-bootstrap-contextmenu/contextMenu.js" ]
                    },
                    {
                        cwd: "<%= globalConfig.importdir %>",
                        src: [ "angular-jwt/dist/angular-jwt.min.js" ],
                    },
                    {
                        cwd: '<%= globalConfig.app %>/bower_components/',
                        src: [ 'flot/jquery.flot.js' ],
                        rename: function () { return 'jquery-flot.js'; }
                    },
                    {
                        cwd: '<%= globalConfig.app %>/bower_components/',
                        src: [ 'flot/jquery.flot.pie.js' ],
                        rename: function () { return 'jquery-flot-pie.js'; }
                    },
                    {
                        cwd: '<%= globalConfig.app %>/bower_components/',
                        src: [ 'flot/jquery.flot.stack.js' ],
                        rename: function () { return 'jquery-flot-stack.js'; }
                    },
                    {
                        cwd: '<%= globalConfig.app %>/bower_components/',
                        src: [ 'flot/jquery.flot.time.js' ],
                        rename: function () { return 'jquery-flot-time.js'; }
                    },
                    {
                        cwd: '<%= globalConfig.app %>/bower_components/',
                        src: [ 'flot.tooltip/js/jquery.flot.tooltip.min.js' ],
                        rename: function () { return 'jquery-flot-tooltip.min.js'; }
                    },
                    {
                        cwd: '<%= globalConfig.app %>/bower_components/',
                        src: [ 'flot-axislabels/jquery.flot.axislabels.js' ],
                        rename: function () { return 'jquery-flot-axislabels.js'; }
                    }
                ]
            }
        }
    },
    'shared-libs-dev': {
        entry: './<%= globalConfig.srcdir %>/scripts/shared-libs.js',
        compile: './<%= globalConfig.resourcesdir %>/assets/js/shared-libs.js',
        options: {
            expose: {
                files: [
                    {
                        cwd: '<%= globalConfig.app %>/bower_components/',
                        src:
                            [
                                'angular-bootstrap/ui-bootstrap.min.js',
                                'angular-bootstrap/ui-bootstrap-tpls.min.js',
                                'angular-loading-bar/build/loading-bar.min.js',
                                'angular-resource/angular-resource.min.js',
                                'angular-route/angular-route.min.js',
                                'angular-sanitize/angular-sanitize.min.js',
                                'angular-ui-router/release/angular-ui-router.min.js',
                                'bootstrap-sass-official/assets/javascripts/bootstrap.min.js',
                                'jquery/jquery.min.js',
                            ]
                    },
                    {
                        cwd: "<%= globalConfig.importdir %>",
                        src: [ "angular/angular.min.js" ]
                    },
                    {
                        cwd: "<%= globalConfig.importdir %>",
                        src: [ "angular-animate/angular-animate.min.js" ]
                    },
                    {
                        cwd: "<%= globalConfig.importdir %>",
                        src: [ "angular-bootstrap-contextmenu/contextMenu.js" ]
                    },
                    {
                        cwd: "<%= globalConfig.importdir %>",
                        src: [ "angular-jwt/dist/angular-jwt.min.js" ],
                    },
                    {
                        cwd: '<%= globalConfig.app %>/bower_components/',
                        src: [ 'flot/jquery.flot.js' ],
                        rename: function () { return 'jquery-flot.js'; }
                    },
                    {
                        cwd: '<%= globalConfig.app %>/bower_components/',
                        src: [ 'flot/jquery.flot.pie.js' ],
                        rename: function () { return 'jquery-flot-pie.js'; }
                    },
                    {
                        cwd: '<%= globalConfig.app %>/bower_components/',
                        src: [ 'flot/jquery.flot.stack.js' ],
                        rename: function () { return 'jquery-flot-stack.js'; }
                    },
                    {
                        cwd: '<%= globalConfig.app %>/bower_components/',
                        src: [ 'flot/jquery.flot.time.js' ],
                        rename: function () { return 'jquery-flot-time.js'; }
                    },
                    {
                        cwd: '<%= globalConfig.app %>/bower_components/',
                        src: [ 'flot.tooltip/js/jquery.flot.tooltip.js' ],
                        rename: function () { return 'jquery-flot-tooltip.js'; }
                    },
                    {
                        cwd: '<%= globalConfig.app %>/bower_components/',
                        src: [ 'flot-axislabels/jquery.flot.axislabels.js' ],
                        rename: function () { return 'jquery-flot-axislabels.js'; }
                    }
                ]
            }
        }
    },
    'app-prod': {
        entry: './<%= globalConfig.srcdir %>/app.js',
        compile: './<%= globalConfig.resourcesdir %>/assets/js/app.js',
        debug: false,
        options: {
            expose: {
                'app-templates':'./<%= globalConfig.tmpdir %>/app-templates.js'
            }
        }
    },
    'app-dev': {
        entry: './<%= globalConfig.srcdir %>/app.js',
        compile: './<%= globalConfig.resourcesdir %>/assets/js/app.js',
        debug: true,
        options: {
            expose: {
                'app-templates':'./<%= globalConfig.tmpdir %>/app-templates.js'
            }
        }
    },
    'app-config': {
        entry: './<%= globalConfig.srcdir %>/scripts/config.js',
        compile: './<%= globalConfig.resourcesdir %>/assets/js/config.js'
    }
};
