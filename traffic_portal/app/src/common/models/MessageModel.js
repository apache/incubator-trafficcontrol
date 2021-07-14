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

var MessageModel = function($rootScope) {

    var model = this;
    var queue = [];

    var messages = {};
    messages.loaded = false;
    messages.content = [];

    this.messages = messages;

    this.setMessages = function(messagesArray, delay) {
        // delay should be true if a redirect follows...
        if (!angular.isArray(messagesArray)) return;
        var messages = {};
        messages.loaded = true;
        messages.content = messagesArray;
        if (delay) {
            queue[0] = messages; // queue up messages to display after a location change
        } else {
            model.messages = messages; // show the messages asap
            queue = []; // clear the queue as messages will be shown immediately
        }
    };

    this.resetMessages = function() {
        messages = {};
        messages.loaded = false;
        messages.content = [];

        this.messages = messages;
    };

    this.removeMessage = function(message) {
        model.messages.content = model.messages.content.filter(function(m) {return m !== message;});
    };

    $rootScope.$on("$locationChangeStart", function() {
        model.resetMessages();
    });

    $rootScope.$on("$locationChangeSuccess", function() {
        model.messages = queue.shift() || {};
    });

};

MessageModel.$inject = ['$rootScope'];
module.exports = MessageModel;
