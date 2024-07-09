package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func allowObjectListWatch(object metav1.ObjectMeta, labelSelector labels.Selector, desiredRv uint64, rvmatch metav1.ResourceVersionMatch) (bool, uint64, error) {
	// rv, err := strconv.ParseUint(object.ResourceVersion, 10, 64)
	// if err != nil {
	// 	return false, 0, err
	// }

	// switch rvmatch {
	// case metav1.ResourceVersionMatchExact:
	// 	if rv != desiredRv {
	// 		return false, 0, nil
	// 	}
	// default:
	// 	if rv < desiredRv {
	// 		return false, 0, nil
	// 	}
	// }

	if labelSelector == nil {
		return true, 1, nil
	}

	if labelSelector.Matches(labels.Set(object.Labels)) {
		return true, 1, nil
	} else {
		return false, 0, nil
	}
}
