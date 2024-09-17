# Changelog

This file lists all notable changes in the project per release. It is
also continously updated with already published but not yet released work.

Release versioning in this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Entries in this file are grouped into several categories:

* Added
* Changed
* Deprecated
* Fixed
* Removed
* Security

## 1.0.0 (2023-11-24)


### Features

* add grpc gw http listener ([c71b20f](https://github.com/opiproject/opi-intel-bridge/commit/c71b20f1c85713148cbce31d6d8cd9a1d186e1b8))
* add grpc interceptor to log calls ([e31be29](https://github.com/opiproject/opi-intel-bridge/commit/e31be295964a5b9fdb54575fcf5d02d94f20965d))
* add otel grpc for monitoring ([7fe8081](https://github.com/opiproject/opi-intel-bridge/commit/7fe8081da8670130f2e690a3fef7b375661becbd))
* **db:** add redis to compose file ([5dbfd9b](https://github.com/opiproject/opi-intel-bridge/commit/5dbfd9b71ed1a1ab6d81223d7c2f43149f9d1795))
* **frontend:** add frontend target api support ([b09a9af](https://github.com/opiproject/opi-intel-bridge/commit/b09a9afe64d054221617c09236a0de7f0f4e4afc))
* **frontend:** check pci port and physical function ([6e665db](https://github.com/opiproject/opi-intel-bridge/commit/6e665db2f3d8d8434a52820151748dc69181db37))
* **frontend:** enable virtio-blk support ([b148dfd](https://github.com/opiproject/opi-intel-bridge/commit/b148dfd47cf04f565d54c6db06ce6c6aa0c6212e))
* **frontend:** pass max number of queues to controller ([7b61beb](https://github.com/opiproject/opi-intel-bridge/commit/7b61bebb04045ab2a70df231e39585d8abd493ed))
* **frontend:** required changes after adding annotations ([5e2e17f](https://github.com/opiproject/opi-intel-bridge/commit/5e2e17f276d8abd8e3cdb27e168fb40ac6c88ff5))
* **frontend:** required changes after adding annotations ([efde344](https://github.com/opiproject/opi-intel-bridge/commit/efde344758580da4d2369d7913a36d6155ce68b1))
* **middleend:** remove unreachable code due to annotations ([a67056d](https://github.com/opiproject/opi-intel-bridge/commit/a67056deb739000198f9b6ec759dbaeb77d35b18))
* **store:** use gokv pkg to abstract persistant store ([5af283c](https://github.com/opiproject/opi-intel-bridge/commit/5af283c67b9211afac5295cd1bf4925c0fd7bbda))
* **store:** use redis ([cd65e70](https://github.com/opiproject/opi-intel-bridge/commit/cd65e7022235f78007d5bae038e0e03355dfb931))


### Bug Fixes

* add PCI device information ([f3c8c3c](https://github.com/opiproject/opi-intel-bridge/commit/f3c8c3c8ee0f7e1a4ce734ee3864db3ce8f0be13))
* **deps:** update github.com/opiproject/gospdk digest to 37c8599 ([221a570](https://github.com/opiproject/opi-intel-bridge/commit/221a570015c397da5c2dff32ec2af2511ff60920))
* **deps:** update github.com/opiproject/gospdk digest to 46d1efd ([145f6f2](https://github.com/opiproject/opi-intel-bridge/commit/145f6f26ad879d969a6a0e01bdc3826b425d1529))
* **deps:** update github.com/opiproject/gospdk digest to 6fe2a5b ([f961407](https://github.com/opiproject/opi-intel-bridge/commit/f9614070ad1cb747ac595fed94ee2614bf2c0dda))
* **deps:** update github.com/opiproject/gospdk digest to 93a4aa9 ([dbf1282](https://github.com/opiproject/opi-intel-bridge/commit/dbf1282090fca383ccd0b22edae9e570e358d73f))
* **deps:** update github.com/opiproject/gospdk digest to d912b55 ([de8f3e8](https://github.com/opiproject/opi-intel-bridge/commit/de8f3e8a6b1dd68970fea43b273ea6ffe2b471ee))
* **deps:** update github.com/opiproject/gospdk digest to de73bd1 ([684115e](https://github.com/opiproject/opi-intel-bridge/commit/684115e3c4e4952b70374e27d7afa01570e4f62d))
* **deps:** update github.com/opiproject/gospdk digest to f4c05ae ([a7eda0c](https://github.com/opiproject/opi-intel-bridge/commit/a7eda0c544e1f5622618a8705a44203a8f7527d3))
* **deps:** update github.com/opiproject/gospdk digest to faeab6c ([230dd21](https://github.com/opiproject/opi-intel-bridge/commit/230dd216052bed219f75346c7a553fab05f9e8fd))
* **deps:** update github.com/opiproject/opi-api digest to 05176c3 ([8e6fcae](https://github.com/opiproject/opi-intel-bridge/commit/8e6fcae7f88edfb3c191e928b0ef1eb7c3f3a1df))
* **deps:** update github.com/opiproject/opi-api digest to 0605e35 ([6ffdf9d](https://github.com/opiproject/opi-intel-bridge/commit/6ffdf9d84b6f0a80c4a1ba356b1b8b8636d498e2))
* **deps:** update github.com/opiproject/opi-api digest to 328ef45 ([c47a22e](https://github.com/opiproject/opi-intel-bridge/commit/c47a22e7bfb6b214357c894e2ab013229ce7cce9))
* **deps:** update github.com/opiproject/opi-api digest to 3ba8d58 ([90e6ad9](https://github.com/opiproject/opi-intel-bridge/commit/90e6ad97fe13b808da97bb95e4328f1c80c25fda))
* **deps:** update github.com/opiproject/opi-api digest to 432a550 ([effd712](https://github.com/opiproject/opi-intel-bridge/commit/effd712899de0e8e91c00d0db37183bf69a05bf7))
* **deps:** update github.com/opiproject/opi-api digest to 4d50cc3 ([be735c8](https://github.com/opiproject/opi-intel-bridge/commit/be735c827dd602b849ebd773112bcb81cb47a116))
* **deps:** update github.com/opiproject/opi-api digest to 520b62d ([d018b4e](https://github.com/opiproject/opi-intel-bridge/commit/d018b4e58138d5f6f6831dab8c9bd10c0e91b2a4))
* **deps:** update github.com/opiproject/opi-api digest to 5b8771b ([edb55ac](https://github.com/opiproject/opi-intel-bridge/commit/edb55ac6fb8ef8f6c5d19dffc2983b7dd6385f04))
* **deps:** update github.com/opiproject/opi-api digest to 625e66a ([ecfbec1](https://github.com/opiproject/opi-intel-bridge/commit/ecfbec1bce9b1ece9a60261291840413a503a61f))
* **deps:** update github.com/opiproject/opi-api digest to 66209d5 ([6a0a817](https://github.com/opiproject/opi-intel-bridge/commit/6a0a817c931a96a82cd0abe4a29ad89c7f635186))
* **deps:** update github.com/opiproject/opi-api digest to 84a85a3 ([7fca39d](https://github.com/opiproject/opi-intel-bridge/commit/7fca39d13b740f4361cdf77e86eaeeb7e05b68c9))
* **deps:** update github.com/opiproject/opi-api digest to 9638639 ([9b9a443](https://github.com/opiproject/opi-intel-bridge/commit/9b9a443a0370aa30bf2983d32f3675ce7d2745d0))
* **deps:** update github.com/opiproject/opi-api digest to 99f5416 ([d5d210a](https://github.com/opiproject/opi-intel-bridge/commit/d5d210acab75a9c60101db527610e6bffa40b728))
* **deps:** update github.com/opiproject/opi-api digest to ab0b6c9 ([4698ff4](https://github.com/opiproject/opi-intel-bridge/commit/4698ff4d02963b83082853782eb267262887296b))
* **deps:** update github.com/opiproject/opi-api digest to b6f178e ([82c4834](https://github.com/opiproject/opi-intel-bridge/commit/82c48345e59e28c7f0708810d12b4fb55c5e1a9f))
* **deps:** update github.com/opiproject/opi-api digest to d8ac77a ([19127aa](https://github.com/opiproject/opi-intel-bridge/commit/19127aad3a156979239cb46ddb0d4c7ad96b2e46))
* **deps:** update github.com/opiproject/opi-api digest to da1d8ce ([5a3da5c](https://github.com/opiproject/opi-intel-bridge/commit/5a3da5c0ed879556cd595ecd9e46b4b25df75f8f))
* **deps:** update github.com/opiproject/opi-api digest to e22215c ([e33f321](https://github.com/opiproject/opi-intel-bridge/commit/e33f321f18fb1e8f61859507a89bddd1e07353d5))
* **deps:** update github.com/opiproject/opi-api digest to e33accd ([7484030](https://github.com/opiproject/opi-intel-bridge/commit/7484030ab2f4ba503e7573feccb54865e86c794c))
* **deps:** update github.com/opiproject/opi-api digest to eb4dd1c ([70fa09d](https://github.com/opiproject/opi-intel-bridge/commit/70fa09d9d8afc23dd9e95810531202166d781150))
* **deps:** update github.com/opiproject/opi-api digest to f1f72ea ([8bb3c23](https://github.com/opiproject/opi-intel-bridge/commit/8bb3c23430afb9cebb9f86e94c739b31ae5bc3df))
* **deps:** update github.com/opiproject/opi-api digest to f31be32 ([f845110](https://github.com/opiproject/opi-intel-bridge/commit/f845110f0ecd70c373a2222a3fb4ae005137237d))
* **deps:** update github.com/opiproject/opi-smbios-bridge digest to 15c2403 ([fb78435](https://github.com/opiproject/opi-intel-bridge/commit/fb78435eaa17f9d90af2e5d003bd40bba70affdf))
* **deps:** update github.com/opiproject/opi-smbios-bridge digest to 4db7394 ([6a36450](https://github.com/opiproject/opi-intel-bridge/commit/6a364508a0a3c7d804818b6277d208b900d6bee8))
* **deps:** update github.com/opiproject/opi-smbios-bridge digest to 535f853 ([bbd0744](https://github.com/opiproject/opi-intel-bridge/commit/bbd0744b9493df912ac31cb6755c59b80fffd643))
* **deps:** update github.com/opiproject/opi-smbios-bridge digest to 6b4e64b ([0b777aa](https://github.com/opiproject/opi-intel-bridge/commit/0b777aac63801ea6ee0cfe99f1f229e1ab8cf0f6))
* **deps:** update github.com/opiproject/opi-smbios-bridge digest to 807413f ([21ac1dc](https://github.com/opiproject/opi-intel-bridge/commit/21ac1dc40505ab7d5872a600d3f729cbf86ad2f5))
* **deps:** update github.com/opiproject/opi-smbios-bridge digest to 8240857 ([a9f6786](https://github.com/opiproject/opi-intel-bridge/commit/a9f6786501f11827501b78903bf2af7d99bd73e7))
* **deps:** update github.com/opiproject/opi-smbios-bridge digest to 9629c69 ([314c133](https://github.com/opiproject/opi-intel-bridge/commit/314c133999a6f802c4803c1d8b12deea56c5b82b))
* **deps:** update github.com/opiproject/opi-smbios-bridge digest to 96fc817 ([8ded554](https://github.com/opiproject/opi-intel-bridge/commit/8ded554d3722cf8b8c391ede4991e8fbe852cdef))
* **deps:** update github.com/opiproject/opi-smbios-bridge digest to acdb4d2 ([43ccbd8](https://github.com/opiproject/opi-intel-bridge/commit/43ccbd8802d5b38e95e17ed2caec03318946e4df))
* **deps:** update github.com/opiproject/opi-smbios-bridge digest to c4f2419 ([80d2232](https://github.com/opiproject/opi-intel-bridge/commit/80d2232bb6b9a5862a6f8d30035fd9c504ae88bf))
* **deps:** update github.com/opiproject/opi-smbios-bridge digest to c627fa6 ([3dc26f1](https://github.com/opiproject/opi-intel-bridge/commit/3dc26f1e253db49bdde9995ca0144f0ff5eabc5d))
* **deps:** update github.com/opiproject/opi-smbios-bridge digest to deb37f3 ([ae8abf5](https://github.com/opiproject/opi-intel-bridge/commit/ae8abf5ed61eb733a5ce846cf821e10035477480))
* **deps:** update github.com/opiproject/opi-smbios-bridge digest to f214b08 ([f7a5106](https://github.com/opiproject/opi-intel-bridge/commit/f7a5106f3f764b363aebdb7f09cc627ce011f1d6))
* **deps:** update github.com/opiproject/opi-smbios-bridge digest to f751cc6 ([fd7189c](https://github.com/opiproject/opi-intel-bridge/commit/fd7189cc3cc7f201211bab2151dd1e348db08fb9))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 012e530 ([1463d1c](https://github.com/opiproject/opi-intel-bridge/commit/1463d1cdd94562a751a6ea9a94b0e5fc4b6a1c2c))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 03aeb07 ([94a998c](https://github.com/opiproject/opi-intel-bridge/commit/94a998c31e6d2db806604d9c5953b8ca89499b8d))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 09c0f6e ([96d7e38](https://github.com/opiproject/opi-intel-bridge/commit/96d7e38669838b966c85898c39b4fe5b710a1c65))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 11e2088 ([08fbce4](https://github.com/opiproject/opi-intel-bridge/commit/08fbce44637a7281814524e80ac82fb0177607b9))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 173f72c ([bb73a33](https://github.com/opiproject/opi-intel-bridge/commit/bb73a33e3e615e573baf8a6c6570b61a65168d04))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 1892538 ([d63423a](https://github.com/opiproject/opi-intel-bridge/commit/d63423a5b161a1c3a6d44b6ffe56bf741e289aef))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 26cce2f ([087765b](https://github.com/opiproject/opi-intel-bridge/commit/087765b436c3257bc33b0d467eefd45912e3dec9))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 2fd8728 ([d538b38](https://github.com/opiproject/opi-intel-bridge/commit/d538b38fe735cbcf8fb8b0393a28772fbeeb4026))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 3887ec9 ([839797e](https://github.com/opiproject/opi-intel-bridge/commit/839797e8fb93dc0092852ea1a3c1e7ce4fdc612a))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 39a2db0 ([209afa0](https://github.com/opiproject/opi-intel-bridge/commit/209afa0b64f4edd734bf3674cd4efb9b53083bd4))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 3c08699 ([cbc3a86](https://github.com/opiproject/opi-intel-bridge/commit/cbc3a86396441d3ff063779fba82f94a266d8a28))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 3cbeab1 ([17f7e89](https://github.com/opiproject/opi-intel-bridge/commit/17f7e897666ebafe0f72918b13adedb0daeac07e))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 3cef6af ([233cc17](https://github.com/opiproject/opi-intel-bridge/commit/233cc170910cbde565f985921f139766cf42a6f9))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 3f9d786 ([9dfe70c](https://github.com/opiproject/opi-intel-bridge/commit/9dfe70c1d1c688ab2fb0e7a103b83e1a5cd3aa12))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 41fa5a6 ([35aa7e4](https://github.com/opiproject/opi-intel-bridge/commit/35aa7e49b25f428b6a5c07e1f8c35fd8790920e3))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 42326c8 ([6ce89c6](https://github.com/opiproject/opi-intel-bridge/commit/6ce89c682602e05c69c0202b73087a3d55961fef))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 448041a ([e4b46e6](https://github.com/opiproject/opi-intel-bridge/commit/e4b46e60c8385043ad6854b02bde0094dd2150f7))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 485a1d2 ([be72a83](https://github.com/opiproject/opi-intel-bridge/commit/be72a8312f260569aa2f5683d26e139c22ee0ccb))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 4cfcc16 ([2710dfc](https://github.com/opiproject/opi-intel-bridge/commit/2710dfc2cd20f7be7eb2b88d71609f4d2c913f3b))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 529538c ([73434a7](https://github.com/opiproject/opi-intel-bridge/commit/73434a72bd2b03049ff3851418381ee3d33d70cc))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 54d9cb7 ([f783eac](https://github.com/opiproject/opi-intel-bridge/commit/f783eac3960c120da8b2a3acca52177017d67b3d))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 5e26cbd ([af61213](https://github.com/opiproject/opi-intel-bridge/commit/af61213df02c1350a7d14021014ecee767fa353d))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 6102354 ([69ae076](https://github.com/opiproject/opi-intel-bridge/commit/69ae07646a74b6f8fc12b62979dec35a06e0fbb8))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 628da04 ([2d560b2](https://github.com/opiproject/opi-intel-bridge/commit/2d560b20bd398bbfc0fd893abf97b16f1fef6a6d))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 6481282 ([d7b53b5](https://github.com/opiproject/opi-intel-bridge/commit/d7b53b5659af3521a8e0856304f147ba44efbee0))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 64ec11e ([8e6552e](https://github.com/opiproject/opi-intel-bridge/commit/8e6552e29e91955e6b224111f6c2eb1346f14102))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 64f20c9 ([5e7ea74](https://github.com/opiproject/opi-intel-bridge/commit/5e7ea7474c6ebf54971cddec13dc9d2b7ba061bd))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 64f8eef ([9323c56](https://github.com/opiproject/opi-intel-bridge/commit/9323c5616a59cf5a731f7131ea04ee7dcaa2cf5b))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 6a8ab66 ([2a2cc2d](https://github.com/opiproject/opi-intel-bridge/commit/2a2cc2d82f27ba798beffd5c890c91075738c788))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 6fba978 ([170e0d0](https://github.com/opiproject/opi-intel-bridge/commit/170e0d007eae8dacda0e032b2416e4a12cb4e51a))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 7455c28 ([3f22629](https://github.com/opiproject/opi-intel-bridge/commit/3f226292b9a0411c06b99d9754d9e71eb65c16f2))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 75888a1 ([80db6d1](https://github.com/opiproject/opi-intel-bridge/commit/80db6d14d98a823caf472d1ea4b40932006b1fb5))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 75da4b5 ([66f38f4](https://github.com/opiproject/opi-intel-bridge/commit/66f38f4477d49babe0ea9b2da24a14492803598f))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 7a5f263 ([ccfce82](https://github.com/opiproject/opi-intel-bridge/commit/ccfce82d553cee542f3768160659220165bfce68))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 7db1105 ([2885654](https://github.com/opiproject/opi-intel-bridge/commit/28856540a53d7a987a3fd26160e08ca61f9b88ed))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 7f180aa ([934675b](https://github.com/opiproject/opi-intel-bridge/commit/934675b262bf6e253697a3a781c16e118a0e4406))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 7f4e06d ([93fdceb](https://github.com/opiproject/opi-intel-bridge/commit/93fdcebc219431bda051ef79b8091c4fbdc50bdc))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 81034be ([e11e63e](https://github.com/opiproject/opi-intel-bridge/commit/e11e63ee2e349eab45d13c72851872d46727cbc2))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 8152570 ([d02f933](https://github.com/opiproject/opi-intel-bridge/commit/d02f9331c95e2a655247c132cd8cee9238f51d9c))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 81f602f ([069cd3b](https://github.com/opiproject/opi-intel-bridge/commit/069cd3bc1f13c4933d5d300c79a74d33d18c3767))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 8249c09 ([97edd0a](https://github.com/opiproject/opi-intel-bridge/commit/97edd0ada57a7cac3c98f31bc462d6378ad1f976))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 8bb4dde ([97a08ad](https://github.com/opiproject/opi-intel-bridge/commit/97a08ad980435256d379b05af2cf5c7a1bc730f3))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 92ff894 ([81d3504](https://github.com/opiproject/opi-intel-bridge/commit/81d3504547ff05d184f7e31ac66f25ade97b5529))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 93d432d ([99139d0](https://github.com/opiproject/opi-intel-bridge/commit/99139d01491c89945b464fbd0ee230bc2d77a778))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 97999f8 ([cd806be](https://github.com/opiproject/opi-intel-bridge/commit/cd806befa92015b6ccd19d377b8983aebcbf4a7f))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 9d299cb ([4f9708c](https://github.com/opiproject/opi-intel-bridge/commit/4f9708c2d7b39317f418bfcae454fef669a47175))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 9d52d66 ([daa5751](https://github.com/opiproject/opi-intel-bridge/commit/daa57515ddb09c00707976a3fb6d6827d4f6e624))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 9ea5135 ([848bccb](https://github.com/opiproject/opi-intel-bridge/commit/848bccb70336c9316bfc271dc0250983aceaabfb))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to 9f244fe ([3373edc](https://github.com/opiproject/opi-intel-bridge/commit/3373edccd55038aa45aedb8aea8563395a75c8b4))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to a18785d ([a464153](https://github.com/opiproject/opi-intel-bridge/commit/a464153d8f166ff7f67138f3f74fa1f7713af5a5))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to a5ffa40 ([fdbc8ad](https://github.com/opiproject/opi-intel-bridge/commit/fdbc8ad6edb332e9b0773500a7ddb8da8b2f0257))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to ab13b2a ([14f3547](https://github.com/opiproject/opi-intel-bridge/commit/14f3547fde8941d116de0bde77505536a22346d4))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to ab2bfc6 ([25db110](https://github.com/opiproject/opi-intel-bridge/commit/25db110755bad8586c616a15eec4b7f8b77d7c55))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to b165b3b ([4e10aa8](https://github.com/opiproject/opi-intel-bridge/commit/4e10aa8bc4560a2928f154c972a64167951e9322))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to b1a5a65 ([10ed50c](https://github.com/opiproject/opi-intel-bridge/commit/10ed50cba43888563b53287b31f90ba73141c4fe))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to b66972a ([b345475](https://github.com/opiproject/opi-intel-bridge/commit/b345475e798cb10715084534f2efdf741abbbae0))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to be3ae1b ([a5daf70](https://github.com/opiproject/opi-intel-bridge/commit/a5daf703e0778dcc4e340dfce8487dfa244321bf))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to be9c5e1 ([eeb9ea1](https://github.com/opiproject/opi-intel-bridge/commit/eeb9ea1cc7c72375618d1befc38e8111e21dc4cf))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to c4fda0e ([dcca342](https://github.com/opiproject/opi-intel-bridge/commit/dcca34282040f9acf4822ac5a228ef540edba675))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to c9569dc ([966dec3](https://github.com/opiproject/opi-intel-bridge/commit/966dec3c1548120859d01992588df435073cbb91))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to ccec4b5 ([afb81bc](https://github.com/opiproject/opi-intel-bridge/commit/afb81bc26e8dbde4c19777168520b0ef37f21c4f))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to d36704d ([25653a7](https://github.com/opiproject/opi-intel-bridge/commit/25653a77136178247177d5702bcf39fa6bd6200f))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to d712068 ([24b4cf8](https://github.com/opiproject/opi-intel-bridge/commit/24b4cf8c86f6d071d42e68818cde431a0f6c4088))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to d876f95 ([796fecc](https://github.com/opiproject/opi-intel-bridge/commit/796fecc575ad9a697eedcdb0727cb04c6aff34b9))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to d88cb4f ([ea25100](https://github.com/opiproject/opi-intel-bridge/commit/ea25100358a4c5a049956f15450d934baca5d8a5))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to d97d54e ([cf76b8a](https://github.com/opiproject/opi-intel-bridge/commit/cf76b8a6e84f8680e6f7cabfb107f84228c9c2f4))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to dd47866 ([00696d4](https://github.com/opiproject/opi-intel-bridge/commit/00696d46820a332a5e8c506c1f8a20b0549a2b3a))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to e285294 ([ad913e1](https://github.com/opiproject/opi-intel-bridge/commit/ad913e1d8678657b6841c3e618edfc476b078db8))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to e3a1aba ([fc0ba45](https://github.com/opiproject/opi-intel-bridge/commit/fc0ba4583ead151cee8825de645f3244b5bd2a10))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to e521fc9 ([7956c90](https://github.com/opiproject/opi-intel-bridge/commit/7956c90caa3edf439175b2e34695a60aba594998))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to e78cbb8 ([89317ca](https://github.com/opiproject/opi-intel-bridge/commit/89317ca5334417a4d06341600a64ad5eaade9882))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to eec6e34 ([176917a](https://github.com/opiproject/opi-intel-bridge/commit/176917a429c8e3a9599243508430fbffaf578ee7))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to f06a8ee ([831dd96](https://github.com/opiproject/opi-intel-bridge/commit/831dd9695a52dfa62227702f403e40ea11b35c10))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to f392c5d ([ee8a47f](https://github.com/opiproject/opi-intel-bridge/commit/ee8a47f4fd801d8e105e2d5ad7bdc0c945b41ca7))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to f5c9729 ([7713d31](https://github.com/opiproject/opi-intel-bridge/commit/7713d318a88f1d33317ec3c4844c5616c8b08dc3))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to f60ea78 ([c319c98](https://github.com/opiproject/opi-intel-bridge/commit/c319c98b44df2f3e2c32aba3e91a20722223a466))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to f78377f ([b78143b](https://github.com/opiproject/opi-intel-bridge/commit/b78143b9143351f34d1c3b81b2d2dba3c2dad948))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to f8b01c6 ([a8159c8](https://github.com/opiproject/opi-intel-bridge/commit/a8159c8a00c995ff177674d9cf339c910bab8e47))
* **deps:** update github.com/opiproject/opi-spdk-bridge digest to f8ecca3 ([6ddbed2](https://github.com/opiproject/opi-intel-bridge/commit/6ddbed295cc813e897f198aa8bd20b5ba9a359e7))
* **deps:** update module google.golang.org/grpc to v1.54.0 ([db2fa84](https://github.com/opiproject/opi-intel-bridge/commit/db2fa84926aa3ca5bd4daae2229db8e44a34a76b))
* **deps:** update module google.golang.org/grpc to v1.55.0 ([66d15d1](https://github.com/opiproject/opi-intel-bridge/commit/66d15d1097ac8918a7a86c0e8a144a5daaaabc89))
* **deps:** update module google.golang.org/protobuf to v1.29.0 ([f9b0600](https://github.com/opiproject/opi-intel-bridge/commit/f9b06005d7e929f3f7358ac37ad99c28ad981e8b))
* **deps:** update module google.golang.org/protobuf to v1.29.1 ([6933675](https://github.com/opiproject/opi-intel-bridge/commit/6933675e7782ba1303dcd27e40c74916e15a5212))
* **deps:** update module google.golang.org/protobuf to v1.30.0 ([2e92901](https://github.com/opiproject/opi-intel-bridge/commit/2e929019df119c7990fc116a9215e1e64d61f806))
* **deps:** update module google.golang.org/protobuf to v1.31.0 ([514aa82](https://github.com/opiproject/opi-intel-bridge/commit/514aa826525daa760ff0dfbe7c90e755bf841d9b))
* **deps:** update module otelgrpc to v0.46.0 ([10e11d2](https://github.com/opiproject/opi-intel-bridge/commit/10e11d28a04bd83b2f5fa082dfd13be6f821cc3f))
* **frontend:** do not allow npi controller for subsys with hostnqn ([1be2642](https://github.com/opiproject/opi-intel-bridge/commit/1be264213d01519a66f85cee034712284afc9d3b))
* **godpu:** new commands introduced ([e64e24f](https://github.com/opiproject/opi-intel-bridge/commit/e64e24f248a94be4b5fc9e2252795d54b6d22fef))
* pass ctx to all Call funcs ([d919904](https://github.com/opiproject/opi-intel-bridge/commit/d919904dbd080879eb35712716f5cfdd3088d13c))
* rename SpdkJSONRPC to Client ([13dd0de](https://github.com/opiproject/opi-intel-bridge/commit/13dd0dec741b930549d3720e3542b2467b3a7daa))
* rename to InventoryService ([343e723](https://github.com/opiproject/opi-intel-bridge/commit/343e72300740b07bd3ae00d7fd019f25cadd80f2))

## [Unreleased]

## [0.2.0] - 2024-01-12

### Added

* Usage of SPDK Acceleration Framework for encryption ([#465](https://github.com/opiproject/opi-intel-bridge/pull/465)).
* Setting number of queues per NVMe controller ([#410](https://github.com/opiproject/opi-intel-bridge/pull/410)).
* Otel monitoring ([#317](https://github.com/opiproject/opi-intel-bridge/pull/317)).
* Nvme/TCP as a target ([#314](https://github.com/opiproject/opi-intel-bridge/pull/314)).
* HTTP gateway for inventory service ([264](https://github.com/opiproject/opi-intel-bridge/pull/264)).
* Virtio-blk support ([#234](https://github.com/opiproject/opi-intel-bridge/pull/234)).

### Security

* Nvme/TCP PSK support ([#318](https://github.com/opiproject/opi-intel-bridge/pull/318)).

## [0.1.0] - 2023-07-14

### Added

* Changelog file ([#167](https://github.com/opiproject/opi-intel-bridge/pull/167)).
* Documentation with usage examples using OPI commands ([#164](https://github.com/opiproject/opi-intel-bridge/pull/164)).
* Enablement of QoS on NVMe device level ([#103](https://github.com/opiproject/opi-intel-bridge/pull/103), [#104](https://github.com/opiproject/opi-intel-bridge/pull/104)).
* Enablement of QoS on volume level ([#92](https://github.com/opiproject/opi-intel-bridge/pull/92)).
* Enablement of HW-accelerated DARE crypto on volume level ([#79](https://github.com/opiproject/opi-intel-bridge/pull/79), [#84](https://github.com/opiproject/opi-intel-bridge/pull/84)).
* Dynamic exposition of NVMe storage devices to the host ([#15](https://github.com/opiproject/opi-intel-bridge/pull/15), [#43](https://github.com/opiproject/opi-intel-bridge/pull/43)).

### Security

* Enablement of mTLS-secured gRPC connection between host and IPU ([#165](https://github.com/opiproject/opi-intel-bridge/pull/165)).
* Hardening of project security through enablement of security scans ([#137](https://github.com/opiproject/opi-intel-bridge/pull/137), [#149](https://github.com/opiproject/opi-intel-bridge/pull/149)).

[unreleased]: https://github.com/opiproject/opi-intel-bridge/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/opiproject/opi-intel-bridge/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/opiproject/opi-intel-bridge/releases/tag/v0.1.0
