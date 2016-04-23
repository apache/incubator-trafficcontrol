var editCameraCalls = {

    editCamera: function (editCameraObj, currentCameraName) {
        var userName = Utilities.getUsername();
        var token = Utilities.getUserToken();

        Meteor.call('editCameraInformation', token, userName, currentCameraName, editCameraObj, function(err, res) {
            if (err) {
                if (err.hasOwnProperty('content')) {
                    alert(JSON.stringify(err.content));
                }
                else {
                    alert(err);
                }
            }
            else {
                if (res.statusCode == 200) {
                    if (res.hasOwnProperty('content')) {
                        res = JSON.parse(res.content);
                        if (res.hasOwnProperty('Status') && res.Status == "Success") {
                            if (res.hasOwnProperty('Message')) { alert(res.Message); }
                            Router.go('browseCameras');
                        }
                    }
                }
                else {
                    alert(JSON.stringify(res.content));
                }
            }
            return res;
        });
    },

    deleteCamera: function(currentCameraName) {
        var userName = Utilities.getUsername();
        var token = Utilities.getUserToken();

        //alert text
        var messageAlert = null;
        var titleAlert = null;
        var typeAlert = null;

        Meteor.call('deleteCamera', token, userName, currentCameraName, function(err, res) {
            if (err) {
                titleAlert = 'Something went wrong!';
                typeAlert = 'error';
                if (err.hasOwnProperty('content')) {
                    messageAlert = JSON.stringify(err.content);
                }
                else {
                    messageAlert = JSON.stringify(err);
                }
            }
            else {
                if (res.statusCode == 200) {
                    typeAlert = 'success';
                    titleAlert = 'Deleted';
                    if (res.hasOwnProperty('content')) {
                        res = JSON.parse(res.content);
                        if (res.hasOwnProperty('Status') && res.Status == "Success") {
                            if (res.hasOwnProperty('Message')) {
                                messageAlert = res.Message;
                            }
                            Router.go('browseCameras');
                        }
                    }
                }
                else {
                    typeAlert = 'error';
                    messageAlert = JSON.stringify(res);
                }
            }
            swal(titleAlert, messageAlert, typeAlert);
            return res;
        });
    }
};

Template.editCamera.events({

    'click #btn-edit-camera': function (evt, tpl) {
        var name = tpl.find('input#edit-cameraname').value;
        var location = tpl.find('input#edit-cameralocation').value;
        var url = tpl.find('input#edit-cameraurl').value;
        var cameraUsername = tpl.find('input#edit-camerausername').value;
        var cameraPassword = tpl.find('input#edit-camerapassword').value;
        var currentCameraName = tpl.find('input#edit-cameraname-current').value;

        if (name && location && url && cameraUsername && cameraPassword && currentCameraName) {
            var cameraObj = {
                name: name,
                location: location,
                url: url,
                username: cameraUsername,
                password: cameraPassword
            };
            editCameraCalls.editCamera(cameraObj, currentCameraName);
        }
        else {
            alert("All fields are required!");
        }
    },

    'click #btn-edit-camera-delete': function(evt, tpl) {

        var currentCameraName = tpl.find('input#edit-cameraname-current').value;

        swal({
            title: "Are you sure?",
            text: "You are about to delete the " + currentCameraName + " camera",
            type: "warning",
            showCancelButton: true,
            confirmButtonColor: "#DD6B55",
            confirmButtonText: "Yes, delete it!",
            closeOnConfirm: false,
            html: false
        }, function(){
            editCameraCalls.deleteCamera(currentCameraName);
        });
    }
});
