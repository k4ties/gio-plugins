//go:build android
// +build android

package hyperlink

import (
	"net/url"
	"sync"

	"git.wow.st/gmp/jni"
)

type driver struct {
	config Config
	mutex  sync.Mutex

	hyperlinkClass      jni.Class
	hyperlinkMethodOpen jni.MethodID
}

func attachDriver(house *Hyperlink, c Config) {
	house.driver = driver{}
	configureDriver(&house.driver, c)
}

func configureDriver(driver *driver, config Config) {
	driver.mutex.Lock()
	defer driver.mutex.Unlock()

	old := driver.config.View
	driver.config = config
	if old != driver.config.View {
		driver.destroy()
		driver.init()
	}
}

func (h *driver) init() {
	jni.Do(jni.JVMFor(h.config.VM), func(env jni.Env) error {
		class, err := jni.LoadClass(env, jni.ClassLoaderFor(env, jni.Object(h.config.Context)), "com/inkeliz/hyperlink_android/hyperlink_android")
		if err != nil {
			panic(err)
		}

		h.hyperlinkClass = jni.Class(jni.NewGlobalRef(env, jni.Object(class)))
		h.hyperlinkMethodOpen = jni.GetStaticMethodID(env, h.hyperlinkClass, "open", "(Landroid/view/View;Ljava/lang/String;Ljava/lang/String;)V")

		return nil
	})
}

func (h *driver) destroy() {
	if h.hyperlinkClass == 0 {
		return
	}
	jni.Do(jni.JVMFor(h.config.VM), func(env jni.Env) error {
		jni.DeleteGlobalRef(env, jni.Object(h.hyperlinkClass))
		h.hyperlinkClass = 0
		h.hyperlinkMethodOpen = nil
		return nil
	})
}

func (h *driver) open(u *url.URL, preferredPackage string) error {
	if h.config.View == 0 {
		return ErrNotReady
	}

	return jni.Do(jni.JVMFor(h.config.VM), func(env jni.Env) error {
		h.mutex.Lock()
		defer h.mutex.Unlock()

		uri := jni.Value(jni.JavaString(env, u.String()))
		pkg := jni.Value(jni.JavaString(env, preferredPackage))

		if err := jni.CallStaticVoidMethod(env, h.hyperlinkClass, h.hyperlinkMethodOpen, jni.Value(h.config.View), uri, pkg); err != nil {
			panic(err)
		}

		return nil
	})
}
