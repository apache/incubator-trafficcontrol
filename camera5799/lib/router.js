Router.configure({
  layoutTemplate: 'layout',
  loadingTemplate: 'loading',
  notFoundTemplate: 'notFound'
});

Router.route('/', {
  name: 'homePage'
});

Router.route('/addCamera/', {
  name: 'addCamera'
});

Router.route('/browseVideos/', {
  name: 'browseVideos'
});

Router.route('/browseCameras/', {
  name: 'browseCameras'
});

Router.route('/cameraDetail/:cameraName/:cameraId', {
  name: 'cameraDetail',
  data: function() {
    console.log("please..." + this.params.cameraName);
    return {
      cameraId: this.params.cameraId,
      cameraName: this.params.cameraName
    };
  }
});