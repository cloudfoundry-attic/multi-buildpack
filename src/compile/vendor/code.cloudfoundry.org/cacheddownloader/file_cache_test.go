package cacheddownloader_test

import (
	"archive/tar"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"code.cloudfoundry.org/cacheddownloader"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("FileCache", func() {
	var (
		cache                     *cacheddownloader.FileCache
		cacheDir                  string
		err                       error
		sourceFile, sourceArchive *os.File
		maxSizeInBytes            int64
		logger                    *lagertest.TestLogger
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		cacheDir, err = ioutil.TempDir("", "cache-test")
		Expect(err).NotTo(HaveOccurred())

		sourceFile = createFile("cache-test-file", "the-file-content")
		sourceArchive = createArchive("cache-test-archive", "Data in Test File")

		maxSizeInBytes = 123424

		cache = cacheddownloader.NewCache(cacheDir, maxSizeInBytes)
	})

	AfterEach(func() {
		os.RemoveAll(sourceFile.Name())
		os.RemoveAll(sourceArchive.Name())
		os.RemoveAll(cacheDir)
	})

	Describe("Add", func() {
		var (
			cacheKey   string
			fileSize   int64
			cacheInfo  cacheddownloader.CachingInfoType
			readCloser *cacheddownloader.CachedFile
		)

		BeforeEach(func() {
			cacheKey = "the-cache-key"
			fileSize = 100
			cacheInfo = cacheddownloader.CachingInfoType{}
		})

		Context("when there is space in the cache", func() {
			JustBeforeEach(func() {
				var err error
				readCloser, err = cache.Add(logger, cacheKey, sourceFile.Name(), fileSize, cacheInfo)
				Expect(err).NotTo(HaveOccurred())
				Expect(readCloser).NotTo(BeNil())
			})

			AfterEach(func() {
				readCloser.Close()
			})

			It("returns a reader", func() {
				content, err := ioutil.ReadAll(readCloser)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(content)).To(Equal("the-file-content"))
				Expect(filenamesInDir(cacheDir)).To(HaveLen(1))
			})

			Context("when closed is called", func() {
				Context("once", func() {
					It("succeeds and has 1 file in the cache", func() {
						Expect(readCloser.Close()).NotTo(HaveOccurred())
						Expect(filenamesInDir(cacheDir)).To(HaveLen(1))
					})
				})

				Context("more than once", func() {
					It("fails", func() {
						Expect(readCloser.Close()).NotTo(HaveOccurred())
						Expect(readCloser.Close()).To(HaveOccurred())
					})
				})
			})

			Context("when a cachekey exists", func() {
				var (
					newSourceFile *os.File
					newFileSize   int64
					newCacheInfo  cacheddownloader.CachingInfoType
					newReader     io.ReadCloser
				)

				BeforeEach(func() {
					newSourceFile = createFile("cache-test-file", "new-file-content")
					newFileSize = fileSize
					newCacheInfo = cacheInfo
				})

				AfterEach(func() {
					os.RemoveAll(newSourceFile.Name())
				})

				Context("when adding the same cache key with identical info", func() {
					It("ignores the add", func() {
						reader, err := cache.Add(logger, cacheKey, newSourceFile.Name(), fileSize, cacheInfo)
						Expect(err).NotTo(HaveOccurred())
						Expect(reader).NotTo(BeNil())
					})
				})

				Context("when a adding the same cache key and different info", func() {
					JustBeforeEach(func() {
						var err error
						newReader, err = cache.Add(logger, cacheKey, newSourceFile.Name(), newFileSize, newCacheInfo)
						Expect(err).NotTo(HaveOccurred())
						Expect(newReader).NotTo(BeNil())
					})

					AfterEach(func() {
						newReader.Close()
					})

					Context("different file size", func() {
						BeforeEach(func() {
							newFileSize = fileSize - 1
						})

						It("returns a reader for the new content", func() {
							content, err := ioutil.ReadAll(newReader)
							Expect(err).NotTo(HaveOccurred())
							Expect(string(content)).To(Equal("new-file-content"))
						})

						It("has files in the cache", func() {
							Expect(filenamesInDir(cacheDir)).To(HaveLen(2))
							Expect(readCloser.Close()).NotTo(HaveOccurred())
							Expect(filenamesInDir(cacheDir)).To(HaveLen(1))
						})

					})

					Context("different caching info", func() {
						BeforeEach(func() {
							newCacheInfo = cacheddownloader.CachingInfoType{
								LastModified: "1234",
							}
						})

						It("returns a reader for the new content", func() {
							content, err := ioutil.ReadAll(newReader)
							Expect(err).NotTo(HaveOccurred())
							Expect(string(content)).To(Equal("new-file-content"))
						})

						It("has files in the cache", func() {
							Expect(filenamesInDir(cacheDir)).To(HaveLen(2))
							Expect(readCloser.Close()).NotTo(HaveOccurred())
							Expect(filenamesInDir(cacheDir)).To(HaveLen(1))
						})

						It("still allows the previous reader to read", func() {
							content, err := ioutil.ReadAll(readCloser)
							Expect(err).NotTo(HaveOccurred())
							Expect(string(content)).To(Equal("the-file-content"))
						})
					})
				})
			})
		})

		Context("when there is not enough space in the cache", func() {
			It("succeeds even if room cannot be allocated", func() {
				var err error
				readCloser, err = cache.Add(logger, cacheKey, sourceFile.Name(), 250000, cacheInfo)
				Expect(err).NotTo(HaveOccurred())
				Expect(readCloser).NotTo(BeNil())
			})
		})
	})

	Describe("AddDirectory", func() {
		var (
			cacheKey, directoryPath string
			fileSize                int64
			cacheInfo               cacheddownloader.CachingInfoType
		)

		BeforeEach(func() {
			cacheKey = "the-cache-key"
			fileSize = 100
			cacheInfo = cacheddownloader.CachingInfoType{}
		})

		Context("when there is space in the cache", func() {
			JustBeforeEach(func() {
				directoryPath, err = cache.AddDirectory(logger, cacheKey, sourceArchive.Name(), fileSize, cacheInfo)
				Expect(err).NotTo(HaveOccurred())
				Expect(directoryPath).NotTo(BeEmpty())
			})

			AfterEach(func() {
				cache.CloseDirectory(logger, cacheKey, directoryPath)
			})

			Context("when the cache is empty", func() {
				It("returns an existing directory", func() {
					fileInfo, err := os.Stat(directoryPath)
					Expect(err).NotTo(HaveOccurred())
					Expect(fileInfo.IsDir()).To(BeTrue())
				})

				It("has 1 file in the cache", func() {
					Expect(filenamesInDir(cacheDir)).To(HaveLen(1))
				})

				It("only accounts for 1 * size of asset", func() {
					entry, ok := cache.Entries[cacheKey]
					Expect(ok).To(BeTrue())
					Expect(entry.Size).To(Equal(fileSize))
				})
			})

			Context("when a cachekey exists", func() {
				var newSourceArchive *os.File
				var newFileSize int64
				var newCacheInfo cacheddownloader.CachingInfoType
				var newDirectoryPath string

				BeforeEach(func() {
					newSourceArchive = createArchive("cache-test-file", "new-file-content")
					newFileSize = fileSize
					newCacheInfo = cacheInfo
				})

				AfterEach(func() {
					os.RemoveAll(newSourceArchive.Name())
				})

				Context("when adding the same cache key with identical info", func() {
					It("ignores the add", func() {
						directoryPath, err := cache.AddDirectory(logger, cacheKey, newSourceArchive.Name(), fileSize, cacheInfo)
						Expect(err).NotTo(HaveOccurred())
						Expect(directoryPath).NotTo(BeEmpty())
					})
				})

				Context("when a adding the same cache key and different info", func() {
					JustBeforeEach(func() {
						var err error
						newDirectoryPath, err = cache.AddDirectory(logger, cacheKey, newSourceArchive.Name(), newFileSize, newCacheInfo)
						Expect(err).NotTo(HaveOccurred())
						Expect(newDirectoryPath).NotTo(BeEmpty())
					})

					AfterEach(func() {
						cache.CloseDirectory(logger, cacheKey, newDirectoryPath)
					})

					Context("different file size", func() {
						BeforeEach(func() {
							newFileSize = fileSize - 1
						})

						It("returns an existing directory", func() {
							fileInfo, err := os.Stat(newDirectoryPath)
							Expect(err).NotTo(HaveOccurred())
							Expect(fileInfo.IsDir()).To(BeTrue())
						})

						It("has files in the cache", func() {
							Expect(filenamesInDir(cacheDir)).To(HaveLen(2))
							cache.CloseDirectory(logger, cacheKey, directoryPath)
							Expect(filenamesInDir(cacheDir)).To(HaveLen(1))
						})
					})

					Context("different caching info", func() {
						BeforeEach(func() {
							newCacheInfo = cacheddownloader.CachingInfoType{
								LastModified: "1234",
							}
						})

						It("returns an existing directory", func() {
							fileInfo, err := os.Stat(newDirectoryPath)
							Expect(err).NotTo(HaveOccurred())
							Expect(fileInfo.IsDir()).To(BeTrue())
						})

						It("has files in the cache", func() {
							Expect(filenamesInDir(cacheDir)).To(HaveLen(2))
							cache.CloseDirectory(logger, cacheKey, directoryPath)
							Expect(filenamesInDir(cacheDir)).To(HaveLen(1))
						})

						It("still allows the previous directory to exist", func() {
							fileInfo, err := os.Stat(directoryPath)
							Expect(err).NotTo(HaveOccurred())
							Expect(fileInfo.IsDir()).To(BeTrue())
						})
					})
				})
			})

			Context("when closed is called", func() {
				Context("once", func() {
					It("succeeds and has 1 file in the cache", func() {
						err = cache.CloseDirectory(logger, cacheKey, directoryPath)
						Expect(err).NotTo(HaveOccurred())
						Expect(filenamesInDir(cacheDir)).To(HaveLen(1))
					})
				})

				Context("more than once", func() {
					It("fails", func() {
						err := cache.CloseDirectory(logger, cacheKey, directoryPath)
						Expect(err).NotTo(HaveOccurred())
						err = cache.CloseDirectory(logger, cacheKey, directoryPath)
						Expect(err).To(HaveOccurred())
					})
				})
			})
		})

		Context("when there is not space in the cache", func() {
			It("succeeds even if room cannot be allocated", func() {
				var err error
				directoryPath, err = cache.AddDirectory(logger, cacheKey, sourceArchive.Name(), 250000, cacheInfo)
				Expect(err).NotTo(HaveOccurred())
				Expect(directoryPath).NotTo(BeEmpty())
			})
		})
	})

	Describe("Get", func() {
		var cacheKey string
		var fileSize int64
		var cacheInfo cacheddownloader.CachingInfoType

		BeforeEach(func() {
			cacheKey = "key"
			fileSize = 100
			cacheInfo = cacheddownloader.CachingInfoType{}
		})

		Context("when there is nothing", func() {
			It("returns nothing", func() {
				reader, ci, err := cache.Get(logger, cacheKey)
				Expect(err).To(Equal(cacheddownloader.EntryNotFound))
				Expect(reader).To(BeNil())
				Expect(ci).To(Equal(cacheInfo))
			})
		})

		Context("when there is an item", func() {
			JustBeforeEach(func() {
				cacheInfo.LastModified = "1234"
				reader, err := cache.Add(logger, cacheKey, sourceFile.Name(), fileSize, cacheInfo)
				Expect(err).NotTo(HaveOccurred())

				Expect(reader.Close()).To(Succeed())
			})

			It("returns a reader for the item", func() {
				reader, ci, err := cache.Get(logger, cacheKey)
				Expect(err).NotTo(HaveOccurred())
				Expect(reader).NotTo(BeNil())
				Expect(ci).To(Equal(cacheInfo))

				content, err := ioutil.ReadAll(reader)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(content)).To(Equal("the-file-content"))
			})

			Context("when the item is replaced", func() {
				var newSourceFile *os.File

				JustBeforeEach(func() {
					newSourceFile = createFile("cache-test-file", "new-file-content")

					cacheInfo.LastModified = "123"
					reader, err := cache.Add(logger, cacheKey, newSourceFile.Name(), fileSize, cacheInfo)
					Expect(err).NotTo(HaveOccurred())
					reader.Close()
				})

				AfterEach(func() {
					os.RemoveAll(newSourceFile.Name())
				})

				It("gets the new item", func() {
					reader, ci, err := cache.Get(logger, cacheKey)
					Expect(err).NotTo(HaveOccurred())
					Expect(reader).NotTo(BeNil())
					Expect(ci).To(Equal(cacheInfo))

					content, err := ioutil.ReadAll(reader)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(content)).To(Equal("new-file-content"))
				})

				Context("when a get is issued before a replace", func() {
					var reader io.ReadCloser
					JustBeforeEach(func() {
						var err error
						reader, _, err = cache.Get(logger, cacheKey)
						Expect(err).NotTo(HaveOccurred())
						Expect(reader).NotTo(BeNil())

						Expect(filenamesInDir(cacheDir)).To(HaveLen(1))
					})

					It("the old file is removed when closed", func() {
						reader.Close()
						Expect(filenamesInDir(cacheDir)).To(HaveLen(1))
					})
				})
			})
		})

		Context("when there is an item added with AddDirectory", func() {
			var (
				dir string
			)

			JustBeforeEach(func() {
				var err error
				cacheInfo.LastModified = "1234"

				_, err = os.Stat(sourceArchive.Name())
				Expect(err).NotTo(HaveOccurred())

				dir, err = cache.AddDirectory(logger, cacheKey, sourceArchive.Name(), fileSize, cacheInfo)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns a reader for the item and keeps the directory", func() {
				reader, ci, err := cache.Get(logger, cacheKey)
				Expect(err).NotTo(HaveOccurred())
				Expect(reader).NotTo(BeNil())
				Expect(ci).To(Equal(cacheInfo))
				Expect(filenamesInDir(cacheDir)).To(HaveLen(2))

				Expect(reader.Name()).To(ContainSubstring(cacheDir))
			})

			It("doubles the size of the entry", func() {
				_, _, err := cache.Get(logger, cacheKey)
				Expect(err).NotTo(HaveOccurred())

				entry, ok := cache.Entries[cacheKey]
				Expect(ok).To(BeTrue())
				Expect(entry.Size).To(Equal(2 * fileSize))
			})

			Context("and the file is closed", func() {
				It("removes the file from the cache and sets the size to 1x", func() {
					reader, _, err := cache.Get(logger, cacheKey)
					Expect(err).NotTo(HaveOccurred())
					Expect(reader).NotTo(BeNil())

					Expect(filenamesInDir(cacheDir)).To(HaveLen(2))
					Expect(reader.Close()).To(Succeed())
					Expect(filenamesInDir(cacheDir)).To(HaveLen(1))

					entry, ok := cache.Entries[cacheKey]
					Expect(ok).To(BeTrue())
					Expect(entry.Size).To(Equal(fileSize))
				})
			})

			Context("and the directory is not in use", func() {
				var (
					reader *cacheddownloader.CachedFile
					ci     cacheddownloader.CachingInfoType
				)

				JustBeforeEach(func() {
					cache.CloseDirectory(logger, cacheKey, dir)
					var err error
					reader, ci, err = cache.Get(logger, cacheKey)
					Expect(err).NotTo(HaveOccurred())
				})

				It("returns a reader for the item and removes the directory", func() {
					Expect(reader).NotTo(BeNil())
					Expect(ci).To(Equal(cacheInfo))
					Expect(filenamesInDir(cacheDir)).To(HaveLen(1))
				})

				It("set the size to 1x", func() {
					entry, ok := cache.Entries[cacheKey]
					Expect(ok).To(BeTrue())
					Expect(entry.Size).To(Equal(fileSize))
				})

				Context("and the directory is later retrieved", func() {
					It("returns a valid path to the directory", func() {
						dir, _, err := cache.GetDirectory(logger, cacheKey)
						Expect(err).NotTo(HaveOccurred())
						Expect(dir).To(BeADirectory())
						Expect(filenamesInDir(cacheDir)).To(HaveLen(2))
					})
				})
			})

			Context("when there is not enough space in the cache", func() {
				BeforeEach(func() {
					maxSizeInBytes = 150
					cache = cacheddownloader.NewCache(cacheDir, maxSizeInBytes)
				})

				JustBeforeEach(func() {
					readCloser, err := cache.Add(logger, "new-cache-key", sourceFile.Name(), 20, cacheddownloader.CachingInfoType{})
					Expect(err).NotTo(HaveOccurred())
					Expect(readCloser.Close()).To(Succeed())
				})

				It("evicts files not in use", func() {
					_, _, err := cache.Get(logger, cacheKey)
					Expect(err).NotTo(HaveOccurred())

					_, _, err = cache.Get(logger, "new-cache-key")
					Expect(err).To(Equal(cacheddownloader.EntryNotFound))
				})
			})
		})
	})

	Describe("GetDirectory", func() {
		var (
			cacheKey  string
			fileSize  int64
			cacheInfo cacheddownloader.CachingInfoType
		)

		BeforeEach(func() {
			cacheKey = "key"
			fileSize = 100
			cacheInfo = cacheddownloader.CachingInfoType{}
		})

		Context("when there is nothing", func() {
			It("returns nothing", func() {
				dir, ci, err := cache.GetDirectory(logger, cacheKey)
				Expect(err).To(Equal(cacheddownloader.EntryNotFound))
				Expect(dir).To(BeEmpty())
				Expect(ci).To(Equal(cacheInfo))
			})
		})

		Context("when there is an added directory", func() {
			BeforeEach(func() {
				cacheInfo.LastModified = "1234"
				dir, err := cache.AddDirectory(logger, cacheKey, sourceArchive.Name(), fileSize, cacheInfo)
				Expect(err).NotTo(HaveOccurred())
				cache.CloseDirectory(logger, cacheKey, dir)
			})

			It("returns a directory path for the item", func() {
				dir, ci, err := cache.GetDirectory(logger, cacheKey)
				Expect(err).NotTo(HaveOccurred())
				Expect(dir).NotTo(BeEmpty())
				Expect(ci).To(Equal(cacheInfo))

				fileInfo, err := os.Stat(dir)
				Expect(err).NotTo(HaveOccurred())
				Expect(fileInfo.IsDir()).To(BeTrue())
				cache.CloseDirectory(logger, cacheKey, dir)
			})

			Context("when the item is replaced", func() {
				var newSourceArchive *os.File

				JustBeforeEach(func() {
					newSourceArchive = createArchive("cache-test-file", "new-file-content")

					cacheInfo.LastModified = "123"
					dir, err := cache.AddDirectory(logger, cacheKey, newSourceArchive.Name(), fileSize, cacheInfo)
					Expect(err).NotTo(HaveOccurred())
					Expect(dir).ToNot(BeEmpty())
					cache.CloseDirectory(logger, cacheKey, dir)
				})

				AfterEach(func() {
					os.RemoveAll(newSourceArchive.Name())
				})

				It("gets the new item", func() {
					dir, ci, err := cache.GetDirectory(logger, cacheKey)
					Expect(err).NotTo(HaveOccurred())
					Expect(dir).NotTo(BeEmpty())
					Expect(ci).To(Equal(cacheInfo))

					fileInfo, err := os.Stat(dir)
					Expect(err).NotTo(HaveOccurred())
					Expect(fileInfo.IsDir()).To(BeTrue())
					cache.CloseDirectory(logger, cacheKey, dir)
				})

				Context("when a get is issued before a replace", func() {
					var dirPath string

					BeforeEach(func() {
						var err error
						dirPath, _, err = cache.GetDirectory(logger, cacheKey)
						Expect(err).NotTo(HaveOccurred())
						Expect(dirPath).NotTo(BeEmpty())

						// Now we need 2 one for the archive and another for the dir
						Expect(filenamesInDir(cacheDir)).To(HaveLen(1))
					})

					It("the old file is removed when closed", func() {
						Expect(filenamesInDir(cacheDir)).To(HaveLen(2))
						cache.CloseDirectory(logger, cacheKey, dirPath)
						Expect(filenamesInDir(cacheDir)).To(HaveLen(1))
					})
				})
			})
		})

		Context("when there is an added archive only", func() {
			var (
				dir           string
				file          *cacheddownloader.CachedFile
				cacheInfoType cacheddownloader.CachingInfoType
				getErr        error
			)

			JustBeforeEach(func() {
				cacheInfo.LastModified = "1234"
				var err error
				file, err = cache.Add(logger, cacheKey, sourceArchive.Name(), fileSize, cacheInfo)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				cache.CloseDirectory(logger, cacheKey, dir)
			})

			It("returns a directory path for the item and leaves the tarball", func() {
				dir, cacheInfoType, getErr = cache.GetDirectory(logger, cacheKey)
				Expect(getErr).NotTo(HaveOccurred())
				Expect(dir).NotTo(BeEmpty())
				Expect(cacheInfoType).To(Equal(cacheInfo))
				Expect(dir).To(BeADirectory())
				Expect(filenamesInDir(cacheDir)).To(HaveLen(2))
			})

			Context("and the directory is closed", func() {
				JustBeforeEach(func() {
					dir, cacheInfoType, getErr = cache.GetDirectory(logger, cacheKey)
					Expect(getErr).NotTo(HaveOccurred())

					err := cache.CloseDirectory(logger, cacheKey, dir)
					Expect(err).NotTo(HaveOccurred())
				})

				It("removes the directory from the cache and sets the size to 1x", func() {
					Expect(filenamesInDir(cacheDir)).To(HaveLen(1))

					entry, ok := cache.Entries[cacheKey]
					Expect(ok).To(BeTrue())
					Expect(entry.Size).To(Equal(fileSize))
				})

				Context("and the directory is later fetched", func() {
					JustBeforeEach(func() {
						dir, cacheInfoType, getErr = cache.GetDirectory(logger, cacheKey)
						Expect(getErr).NotTo(HaveOccurred())
						Expect(dir).NotTo(BeEmpty())
						Expect(cacheInfoType).To(Equal(cacheInfo))
					})

					It("returns a valid path", func() {
						Expect(dir).To(BeADirectory())
						Expect(filenamesInDir(cacheDir)).To(HaveLen(2))
					})

					It("has the contents of the archive", func() {
						paths := recursiveList(dir)
						Expect(paths).To(ConsistOf(
							"/bin",
							"/bin/readme.txt",
							"/diego.txt",
							"/testdir",
							"/testdir/file.txt",
							"/testdir/fileLink.txt",
						))
					})

					It("correctly expands symbolic links", func() {
						fullpath := filepath.Join(dir, "/testdir/fileLink.txt")
						fileInfo, err := os.Lstat(fullpath)
						Expect(err).NotTo(HaveOccurred())
						Expect(fileInfo.Mode().IsRegular()).ToNot(BeTrue())
						Expect(fileInfo.Mode() & os.ModeSymlink).To(Equal(os.ModeSymlink))
					})

				})
			})

			Context("when the file is not in use and the directory is fetched", func() {
				JustBeforeEach(func() {
					Expect(file.Close()).To(Succeed())
					dir, cacheInfoType, getErr = cache.GetDirectory(logger, cacheKey)
					Expect(getErr).NotTo(HaveOccurred())
					Expect(dir).NotTo(BeEmpty())
					Expect(cacheInfoType).To(Equal(cacheInfo))
				})

				It("returns a directory path for the item and removes the tarball", func() {
					Expect(dir).To(BeADirectory())
					Expect(filenamesInDir(cacheDir)).To(HaveLen(1))
				})

				Context("after a compaction", func() {
					var tempFile string

					JustBeforeEach(func() {
						By("compacting the directory")
						file, _, err = cache.Get(logger, cacheKey)
						Expect(err).NotTo(HaveOccurred())
					})

					AfterEach(func() {
						Expect(os.RemoveAll(tempFile)).To(Succeed())
					})

					It("returns a reader that is valid tar archive", func() {
						reader := tar.NewReader(file)
						_, err := reader.Next()
						Expect(err).NotTo(HaveOccurred())
					})
				})

				Context("after an iteration of compaction/extraction", func() {
					JustBeforeEach(func() {
						By("compacting the directory")
						file, _, err = cache.Get(logger, cacheKey)
						Expect(err).NotTo(HaveOccurred())
						Expect(cache.CloseDirectory(logger, cacheKey, dir)).To(Succeed())
						Expect(file.Close()).To(Succeed())

						By("extracting the tar file")
						dir, _, err = cache.GetDirectory(logger, cacheKey)
					})

					It("contains the same content", func() {
						paths := recursiveList(dir)
						Expect(paths).To(ConsistOf(
							"/bin",
							"/bin/readme.txt",
							"/diego.txt",
							"/testdir",
							"/testdir/file.txt",
							"/testdir/fileLink.txt",
						))
					})
				})
			})

			Context("when there isn't enough space", func() {
				BeforeEach(func() {
					maxSizeInBytes = 10
					cache = cacheddownloader.NewCache(cacheDir, maxSizeInBytes)
				})

				JustBeforeEach(func() {
					dir, cacheInfoType, getErr = cache.GetDirectory(logger, cacheKey)
				})

				It("should not return an error", func() {
					Expect(getErr).NotTo(HaveOccurred())
				})

				Context("and all references to the file are closed", func() {
					JustBeforeEach(func() {
						file.Close()
						Expect(cache.CloseDirectory(logger, cacheKey, dir)).To(Succeed())
					})

					It("should delete the directory as soon as we add another file to the cache", func() {
						newArchive := createArchive("cache-test-file", "new-file-content")

						_, err := cache.Add(logger, "new-cache-key", newArchive.Name(), fileSize, cacheInfo)
						Expect(err).NotTo(HaveOccurred())

						Expect(dir).NotTo(BeADirectory())
					})
				})
			})
		})
	})

	Describe("Remove", func() {
		var (
			cacheKey  string
			cacheInfo cacheddownloader.CachingInfoType
		)

		Context("when doing Add", func() {
			JustBeforeEach(func() {
				cacheKey = "key"

				cacheInfo.LastModified = "1234"
				reader, err := cache.Add(logger, cacheKey, sourceFile.Name(), 100, cacheInfo)
				Expect(err).NotTo(HaveOccurred())
				Expect(reader.Close()).To(Succeed())
			})

			Context("when the key does not exist", func() {
				It("does not fail", func() {
					Expect(func() { cache.Remove(logger, "bogus") }).NotTo(Panic())
				})
			})

			Context("when the key exists", func() {
				It("removes the file in the cache", func() {
					Expect(filenamesInDir(cacheDir)).To(HaveLen(1))
					cache.Remove(logger, cacheKey)
					Expect(filenamesInDir(cacheDir)).To(HaveLen(0))
				})
			})

			Context("when a get is issued first", func() {
				It("removes the file after a close", func() {
					reader, _, err := cache.Get(logger, cacheKey)
					Expect(err).NotTo(HaveOccurred())
					Expect(reader).NotTo(BeNil())
					Expect(filenamesInDir(cacheDir)).To(HaveLen(1))

					cache.Remove(logger, cacheKey)
					Expect(filenamesInDir(cacheDir)).To(HaveLen(1))

					reader.Close()
					Expect(filenamesInDir(cacheDir)).To(HaveLen(0))
				})
			})
		})

		Context("when doing AddDirectory add", func() {
			JustBeforeEach(func() {
				cacheKey = "key"

				cacheInfo.LastModified = "1234"
				dir, err := cache.AddDirectory(logger, cacheKey, sourceArchive.Name(), 100, cacheInfo)
				Expect(err).NotTo(HaveOccurred())
				cache.CloseDirectory(logger, cacheKey, dir)
			})

			Context("when the key does not exist", func() {
				It("does not fail", func() {
					Expect(func() { cache.Remove(logger, "bogus") }).NotTo(Panic())
				})
			})

			Context("when the key exists", func() {
				It("removes the file in the cache", func() {
					Expect(filenamesInDir(cacheDir)).To(HaveLen(1))
					cache.Remove(logger, cacheKey)
					Expect(filenamesInDir(cacheDir)).To(HaveLen(0))
				})
			})

			Context("when a get is issued first", func() {
				It("removes the file after a close", func() {
					dir, _, err := cache.GetDirectory(logger, cacheKey)
					Expect(err).NotTo(HaveOccurred())
					Expect(dir).NotTo(BeEmpty())
					Expect(filenamesInDir(cacheDir)).To(HaveLen(1))

					cache.Remove(logger, cacheKey)
					Expect(filenamesInDir(cacheDir)).To(HaveLen(1))

					cache.CloseDirectory(logger, cacheKey, dir)
					Expect(filenamesInDir(cacheDir)).To(HaveLen(0))
				})
			})
		})
	})
})

func createFile(filename string, content string) *os.File {
	sourceFile, err := ioutil.TempFile("", filename)
	Expect(err).NotTo(HaveOccurred())
	sourceFile.WriteString(content)
	sourceFile.Close()

	return sourceFile
}

func createArchive(filename, sampleData string) *os.File {
	sourceFile, err := ioutil.TempFile("", filename)

	Expect(err).NotTo(HaveOccurred())

	// Create a new tar archive.
	tw := tar.NewWriter(sourceFile)

	// Add some files to the archive.
	var files = []struct {
		Name, Body string
		Type       byte
		Mode       int64
	}{
		{"bin/readme.txt", "This archive contains some text files.", tar.TypeReg, 0600},
		{"diego.txt", "Diego names:\nVizzini\nGeoffrey\nPrincess Buttercup\n", tar.TypeReg, 0600},
		{"testdir", "", tar.TypeDir, 0766},
		{"testdir/file.txt", sampleData, tar.TypeReg, 0600},
		{"testdir/fileLink.txt", "/diego.txt", tar.TypeSymlink, 0600},
	}
	for _, file := range files {
		var hdr *tar.Header
		if file.Type == tar.TypeSymlink {
			hdr = &tar.Header{
				Name:     file.Name,
				Typeflag: file.Type,
				Mode:     file.Mode,
				Linkname: file.Body,
			}
			hdr.Mode &^= 040000 // c_ISDIR
			hdr.Mode |= 0120000 // c_ISLNK
		} else {
			hdr = &tar.Header{
				Name:     file.Name,
				Typeflag: file.Type,
				Mode:     file.Mode,
				Size:     int64(len(file.Body)),
			}
		}
		err := tw.WriteHeader(hdr)
		Expect(err).NotTo(HaveOccurred())
		if file.Type != tar.TypeSymlink {
			_, err = tw.Write([]byte(file.Body))
			Expect(err).NotTo(HaveOccurred())
		}
	}
	// Make sure to check the error on Close.
	err = tw.Close()
	Expect(err).NotTo(HaveOccurred())
	err = sourceFile.Close()
	Expect(err).NotTo(HaveOccurred())
	return sourceFile
}

func filenamesInDir(dir string) []string {
	entries, err := ioutil.ReadDir(dir)
	Expect(err).NotTo(HaveOccurred())

	result := []string{}
	for _, entry := range entries {
		result = append(result, entry.Name())
	}

	return result
}

func recursiveList(dir string) []string {
	paths := []string{}
	filepath.Walk(dir, func(path string, _ os.FileInfo, _ error) error {
		path = strings.TrimPrefix(path, dir)
		if path != "" {
			paths = append(paths, filepath.ToSlash(path))
		}
		return nil
	})
	return paths
}
