mciModule.controller('PatchController', function($scope, $filter, $window, notificationService, $http) {
  $scope.userTz = $window.userTz;
  $scope.canEdit = $window.canEdit;
  var checkedProp = _.property("checked")

  $scope.selectVariant = function($event, index){
    $event.preventDefault()
    if ($event.ctrlKey || $event.metaKey) {
      // Ctrl/Meta+Click: Toggle just the variant being clicked.
      $scope.variants[index].checked = !$scope.variants[index].checked
    } else if ($event.shiftKey) {
      // Shift+Click: Select everything between the first element 
      // that's already selected element and the element being clicked on.
      var firstCheckedIndex = _.findIndex($scope.variants, checkedProp)
      firstCheckedIndex = Math.max(firstCheckedIndex, 0) // if nothing selected yet, start at 0.
      var indexBounds = Array(firstCheckedIndex, index).sort(function(a, b){
        return a-b;
      })
      for(var i=indexBounds[0]; i<=indexBounds[1]; i++){
        $scope.variants[i].checked = true
      }
    } else {
      // Regular click: Select *only* the one being clicked, and unselect all others.
      for(var i=0; i<$scope.variants.length;i++){
        $scope.variants[i].checked = (i == index)
      }
    }
  }

  $scope.numSetForVariant = function(variantId){
    var v = _.find($scope.variants, function(x){return x.id == variantId})
    return _.filter(_.pluck(v.tasks, "checked"), _.identity).length
  }

  $scope.selectedVariants = function(){
    return _.filter($scope.variants, checkedProp)
  }

  $scope.getActiveTasks = function(){
    var selectedVariants = $scope.selectedVariants()

    // return the union of the set of tasks shared by all of them, sorted by name
    var tasksInSelectedVariants = _.uniq(_.flatten(_.map(_.pluck(selectedVariants, "tasks"), _.keys)))
    return tasksInSelectedVariants.sort()
  }

  $scope.selectedTasksByVariant = function(variantId){
    return _.filter(_.find($scope.variants, function(x){return x.id == varinatId}).tasks, _.property("selected"))
  }

  $scope.changeStateAll = function(state){
    var selectedVariantNames = _.object(_.map(_.pluck($scope.selectedVariants(), "id"), function(id){return [id, true]}))
    var activeTasks = $scope.getActiveTasks()
    for(var i=0;i<$scope.variants.length;i++){
      var v = $scope.variants[i];
      if(!(v.id in selectedVariantNames)){
        continue;
      }
      _.each(activeTasks, function(taskName){
        v.tasks[taskName].checked = newValue;
      })
    }
  }

  $scope.save = function(){
    var data = _.map($scope.variants, function(v){
      return {
        variant: v.id, 
        tasks: _.keys(_.omit(v.tasks, function(v){return !v.checked})),
      };
    })
    $http.post('/patch/' + $scope.patch.Id, data).
      success(function(data, status) {
        window.location.replace("/version/" + data.version);
      }).
      error(function(data, status, errorThrown) {
      	notificationService.pushNotification('Error retrieving logs: ' + JSON.stringify(data), 'errorHeader');
      });
  };

  $scope.setPatchInfo = function() {
    $scope.patch = $window.patch;
    $scope.patchContainer = {'Patch':$scope.patch}
    var patch = $scope.patch;

    $scope.variants = _.map($window.variants, function(v, variantId){
      return {
        id: variantId, 
        checked:false,
        name: v.DisplayName,
        tasks : _.object(_.map(_.pluck(v.Tasks, "Name"), function(t){
          return [t, {checked:false}]
        }))
      };
    })

    var allUniqueTaskNames = _.uniq(_.flatten(_.map(_.pluck($scope.variants, "tasks"), _.keys)))

    $scope.tasks = _.object(_.map(allUniqueTaskNames, function(taskName){
      // create a getter/setter for the state of the task
      return [taskName, function(newValue){
        var selectedVariants = $scope.selectedVariants()
        if(!arguments.length){ // called with no args, act as a getter
          var statusAcrossVariants = _.flatten(_.map(_.pluck($scope.selectedVariants(), "tasks"), function(o){return _.filter(o, function(v, k){return k==taskName})}))
          var groupCountedStatus = _.countBy(statusAcrossVariants, function(x){return x.checked == true})
          if(groupCountedStatus[true] == statusAcrossVariants.length ){
            return true
          }else if(groupCountedStatus[false] == statusAcrossVariants.length ){
            return false
          }
          return null;
        }

        var selectedVariantNames = _.object(_.map(_.pluck(selectedVariants, "id"), function(id){return [id, true]}))
        
        // act as a setter
        for(var i=0;i<$scope.variants.length;i++){
          var v = $scope.variants[i];
          if(!(v.id in selectedVariantNames)){
            continue;
          }
          if(_.has(v.tasks, taskName)){
            v.tasks[taskName].checked = newValue;
          }
        }
        return newValue
      }];
    }))
  }

  $scope.setPatchInfo();
})
